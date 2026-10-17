package com.pizza.auth.service;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doNothing;
import static org.mockito.Mockito.doThrow;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.lang.reflect.Field;
import java.util.Optional;
import java.util.UUID;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.security.crypto.password.PasswordEncoder;

import com.pizza.auth.dto.ChangePasswordRequest;
import com.pizza.auth.dto.RegisterRequest;
import com.pizza.auth.dto.RegisterResponse;
import com.pizza.auth.dto.UserLookupResponse;
import com.pizza.auth.dto.UserResponse;
import com.pizza.auth.exception.EmailAlreadyExistsException;
import com.pizza.auth.exception.InvalidCurrentPasswordException;
import com.pizza.auth.exception.InvalidTokenException;
import com.pizza.auth.exception.UsernameAlreadyExistsException;
import com.pizza.auth.security.RefreshTokenService;
import com.pizza.auth.security.VerificationTokenService;
import com.pizza.auth.user.AccountStatus;
import com.pizza.auth.user.User;
import com.pizza.auth.user.UserRepository;

@ExtendWith(MockitoExtension.class)
class UserServiceTest {

    @Mock
    private UserRepository userRepository;
    @Mock
    private PasswordEncoder passwordEncoder;
    @Mock
    private VerificationTokenService verificationTokenService;
    @Mock
    private EmailService emailService;
    @Mock
    private RefreshTokenService refreshTokenService;

    @InjectMocks
    private UserServiceImpl userService;

    private RegisterRequest request;

    @BeforeEach
    void setUp() {
        request = new RegisterRequest();
        request.setEmail("Test.User@Example.COM");
        request.setUsername("test_user");
        request.setPassword("Str0ngP@ss!");
        request.setFullName("Nguyen Van A");
    }

    @Test
    void register_success() throws Exception {
        when(userRepository.existsByEmail("test.user@example.com")).thenReturn(false);
        when(userRepository.existsByUsername("test_user")).thenReturn(false);
        when(passwordEncoder.encode("Str0ngP@ss!")).thenReturn("hashed-password");
        when(userRepository.saveAndFlush(any(User.class))).thenAnswer(invocation -> {
            User u = invocation.getArgument(0);
            Field f = User.class.getDeclaredField("id");
            f.setAccessible(true);
            f.set(u, UUID.randomUUID());
            return u;
        });
        when(verificationTokenService.create(any(UUID.class))).thenReturn("verify-token");
        doNothing().when(emailService).sendVerificationEmail(anyString(), anyString());

        RegisterResponse response = userService.register(request);

        assertThat(response.getEmail()).isEqualTo("test.user@example.com");
        assertThat(response.getUsername()).isEqualTo("test_user");
        assertThat(response.getFullName()).isEqualTo("Nguyen Van A");
        assertThat(response.getStatus()).isEqualTo(AccountStatus.PENDING_VERIFICATION);
        verify(emailService).sendVerificationEmail(eq("test.user@example.com"), eq("verify-token"));
        verify(userRepository).saveAndFlush(any(User.class));
    }

    @Test
    void register_emailAlreadyExists_throws() {
        when(userRepository.existsByEmail("test.user@example.com")).thenReturn(true);
        assertThatThrownBy(() -> userService.register(request))
                .isInstanceOf(EmailAlreadyExistsException.class);
        verify(userRepository, never()).saveAndFlush(any());
    }

    @Test
    void register_usernameAlreadyExists_throws() {
        when(userRepository.existsByEmail("test.user@example.com")).thenReturn(false);
        when(userRepository.existsByUsername("test_user")).thenReturn(true);
        assertThatThrownBy(() -> userService.register(request))
                .isInstanceOf(UsernameAlreadyExistsException.class);
        verify(userRepository, never()).saveAndFlush(any());
    }

    @Test
    void register_mailFailure_stillSucceeds() throws Exception {
        when(userRepository.existsByEmail("test.user@example.com")).thenReturn(false);
        when(userRepository.existsByUsername("test_user")).thenReturn(false);
        when(passwordEncoder.encode("Str0ngP@ss!")).thenReturn("hashed-password");
        when(userRepository.saveAndFlush(any(User.class))).thenAnswer(invocation -> {
            User u = invocation.getArgument(0);
            Field f = User.class.getDeclaredField("id");
            f.setAccessible(true);
            f.set(u, UUID.randomUUID());
            return u;
        });
        when(verificationTokenService.create(any(UUID.class))).thenReturn("verify-token");
        doThrow(new RuntimeException("smtp down")).when(emailService)
                .sendVerificationEmail(anyString(), anyString());

        RegisterResponse response = userService.register(request);
        assertThat(response.getEmail()).isEqualTo("test.user@example.com");
        assertThat(response.getUsername()).isEqualTo("test_user");
    }

    @Test
    void verifyEmail_success() {
        UUID userId = UUID.randomUUID();
        User user = new User("user@example.com", "testuser", "hash", "Name");
        when(verificationTokenService.consume("good-token")).thenReturn(Optional.of(userId));
        when(userRepository.findById(userId)).thenReturn(Optional.of(user));
        when(userRepository.save(any(User.class))).thenAnswer(i -> i.getArgument(0));

        userService.verifyEmail("good-token");

        assertThat(user.getStatus()).isEqualTo(AccountStatus.ACTIVE);
        assertThat(user.getEmailVerifiedAt()).isNotNull();
    }

    @Test
    void verifyEmail_notPending_noChange() {
        UUID userId = UUID.randomUUID();
        User user = new User("user@example.com", "testuser", "hash", "Name");
        user.setStatus(AccountStatus.DISABLED);
        when(verificationTokenService.consume("good-token")).thenReturn(Optional.of(userId));
        when(userRepository.findById(userId)).thenReturn(Optional.of(user));

        userService.verifyEmail("good-token");

        assertThat(user.getStatus()).isEqualTo(AccountStatus.DISABLED);
        verify(userRepository, never()).save(any());
    }

    @Test
    void verifyEmail_invalidToken_throws() {
        when(verificationTokenService.consume("bad")).thenReturn(Optional.empty());
        assertThatThrownBy(() -> userService.verifyEmail("bad"))
                .isInstanceOf(InvalidTokenException.class);
    }

    @Test
    void getCurrentUser_success() {
        UUID userId = UUID.randomUUID();
        User user = new User("user@example.com", "testuser", "hash", "Name");
        when(userRepository.findById(userId)).thenReturn(Optional.of(user));

        UserResponse response = userService.getCurrentUser(userId);

        assertThat(response.getEmail()).isEqualTo("user@example.com");
        assertThat(response.getUsername()).isEqualTo("testuser");
        assertThat(response.getFullName()).isEqualTo("Name");
    }

    @Test
    void lookup_success() {
        User user = new User("user@example.com", "testuser", "hash", "Name");
        when(userRepository.findByEmailOrUsername("testuser")).thenReturn(Optional.of(user));

        UserLookupResponse response = userService.lookup("testuser");

        assertThat(response.getEmail()).isEqualTo("user@example.com");
        assertThat(response.getUsername()).isEqualTo("testuser");
        assertThat(response.getFullName()).isEqualTo("Name");
    }

    @Test
    void lookup_notFound_throws() {
        when(userRepository.findByEmailOrUsername("missing")).thenReturn(Optional.empty());
        assertThatThrownBy(() -> userService.lookup("missing"))
                .isInstanceOf(InvalidTokenException.class);
    }

    @Test
    void changePassword_success() {
        UUID userId = UUID.randomUUID();
        User user = new User("user@example.com", "testuser", "old-hash", "Name");
        when(userRepository.findById(userId)).thenReturn(Optional.of(user));
        when(passwordEncoder.matches("oldPass", "old-hash")).thenReturn(true);
        when(passwordEncoder.encode("newPass")).thenReturn("new-hash");
        when(userRepository.save(any(User.class))).thenAnswer(i -> i.getArgument(0));
        doNothing().when(refreshTokenService).revokeAllForUser(userId);

        ChangePasswordRequest req = new ChangePasswordRequest();
        req.setCurrentPassword("oldPass");
        req.setNewPassword("newPass");
        userService.changePassword(userId, req);

        assertThat(user.getPasswordHash()).isEqualTo("new-hash");
        verify(refreshTokenService).revokeAllForUser(userId);
    }

    @Test
    void changePassword_wrongCurrent_throws() {
        UUID userId = UUID.randomUUID();
        User user = new User("user@example.com", "testuser", "old-hash", "Name");
        when(userRepository.findById(userId)).thenReturn(Optional.of(user));
        when(passwordEncoder.matches("wrong", "old-hash")).thenReturn(false);

        ChangePasswordRequest req = new ChangePasswordRequest();
        req.setCurrentPassword("wrong");
        req.setNewPassword("newPass");
        assertThatThrownBy(() -> userService.changePassword(userId, req))
                .isInstanceOf(InvalidCurrentPasswordException.class);
    }
}
