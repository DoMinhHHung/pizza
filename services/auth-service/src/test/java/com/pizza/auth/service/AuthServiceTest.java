package com.pizza.auth.service;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doNothing;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import com.pizza.auth.config.JwtProperties;
import com.pizza.auth.config.SecurityProperties;
import com.pizza.auth.dto.ForgotPasswordRequest;
import com.pizza.auth.dto.LoginRequest;
import com.pizza.auth.dto.ResetPasswordRequest;
import com.pizza.auth.dto.TokenResponse;
import com.pizza.auth.exception.AccountLockedException;
import com.pizza.auth.exception.InvalidCredentialsException;
import com.pizza.auth.exception.InvalidTokenException;
import com.pizza.auth.security.JwtService;
import com.pizza.auth.security.LoginAttemptService;
import com.pizza.auth.security.PasswordResetTokenService;
import com.pizza.auth.security.RefreshTokenService;
import com.pizza.auth.security.TokenBlacklistService;
import com.pizza.auth.user.AccountStatus;
import com.pizza.auth.user.User;
import com.pizza.auth.user.UserRepository;
import java.time.Duration;
import java.util.Optional;
import java.util.UUID;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.Mockito;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.security.crypto.password.PasswordEncoder;

@ExtendWith(MockitoExtension.class)
class AuthServiceTest {

    @Mock private UserRepository userRepository;
    @Mock private PasswordEncoder passwordEncoder;
    @Mock private JwtService jwtService;
    @Mock private RefreshTokenService refreshTokenService;
    @Mock private TokenBlacklistService tokenBlacklistService;
    @Mock private LoginAttemptService loginAttemptService;
    @Mock private PasswordResetTokenService passwordResetTokenService;
    @Mock private EmailService emailService;
    @Mock private JwtProperties jwtProperties;
    @Mock private SecurityProperties securityProperties;

    @InjectMocks
    private AuthServiceImpl authService;

    private User activeUser;
    private LoginRequest loginRequest;

    @BeforeEach
    void setUp() {
        activeUser = new User("user@example.com", "hashed", "Test User");
        activeUser.setStatus(AccountStatus.ACTIVE);
        loginRequest = new LoginRequest();
        loginRequest.setEmail("user@example.com");
        loginRequest.setPassword("password");
        Mockito.lenient().when(jwtProperties.getAccessTokenExpiration()).thenReturn(Duration.ofMinutes(30));
        Mockito.lenient().when(jwtProperties.getRefreshTokenExpiration()).thenReturn(Duration.ofDays(7));
    }

    @Test
    void login_success() {
        when(loginAttemptService.isLocked("user@example.com")).thenReturn(false);
        when(userRepository.findByEmail("user@example.com")).thenReturn(Optional.of(activeUser));
        when(passwordEncoder.matches("password", "hashed")).thenReturn(true);
        when(jwtService.generateAccessToken(any(), anyString())).thenReturn("access");
        when(jwtService.generateRefreshToken(any(), anyString())).thenReturn("refresh");
        doNothing().when(refreshTokenService).store(any(), anyString(), anyString());
        doNothing().when(loginAttemptService).clearFailures(anyString());

        TokenResponse response = authService.login(loginRequest);

        assertThat(response.getAccessToken()).isEqualTo("access");
        assertThat(response.getExpiresIn()).isEqualTo(1800L);
        verify(loginAttemptService).clearFailures("user@example.com");
    }

    @Test
    void login_wrongPassword_recordsFailure() {
        when(loginAttemptService.isLocked("user@example.com")).thenReturn(false);
        when(userRepository.findByEmail("user@example.com")).thenReturn(Optional.of(activeUser));
        when(passwordEncoder.matches("password", "hashed")).thenReturn(false);
        when(loginAttemptService.isLocked("user@example.com")).thenReturn(false);

        assertThatThrownBy(() -> authService.login(loginRequest))
                .isInstanceOf(InvalidCredentialsException.class);
        verify(loginAttemptService).recordFailure("user@example.com");
    }

    @Test
    void login_lockedByAttempts_throws() {
        when(loginAttemptService.isLocked("user@example.com")).thenReturn(true);
        assertThatThrownBy(() -> authService.login(loginRequest))
                .isInstanceOf(AccountLockedException.class);
        verify(userRepository, never()).findByEmail(anyString());
    }

    @Test
    void forgotPassword_existingUser_sendsEmail() {
        when(userRepository.findByEmail("user@example.com")).thenReturn(Optional.of(activeUser));
        when(passwordResetTokenService.create(any())).thenReturn("reset-token");
        doNothing().when(emailService).sendPasswordResetEmail(anyString(), anyString());

        ForgotPasswordRequest req = new ForgotPasswordRequest();
        req.setEmail("user@example.com");
        authService.forgotPassword(req);

        verify(emailService).sendPasswordResetEmail(eq("user@example.com"), eq("reset-token"));
    }

    @Test
    void forgotPassword_unknownEmail_noOp() {
        when(userRepository.findByEmail("unknown@example.com")).thenReturn(Optional.empty());
        ForgotPasswordRequest req = new ForgotPasswordRequest();
        req.setEmail("unknown@example.com");
        authService.forgotPassword(req);
        verify(emailService, never()).sendPasswordResetEmail(anyString(), anyString());
    }

    @Test
    void resetPassword_success() {
        UUID userId = UUID.randomUUID();
        when(passwordResetTokenService.consume("good-token")).thenReturn(Optional.of(userId));
        when(userRepository.findById(userId)).thenReturn(Optional.of(activeUser));
        when(passwordEncoder.encode("newPass")).thenReturn("new-hash");
        when(userRepository.save(any())).thenAnswer(i -> i.getArgument(0));
        doNothing().when(refreshTokenService).revokeAllForUser(userId);
        doNothing().when(loginAttemptService).clearFailures(anyString());

        ResetPasswordRequest req = new ResetPasswordRequest();
        req.setToken("good-token");
        req.setNewPassword("newPass");
        authService.resetPassword(req);

        assertThat(activeUser.getPasswordHash()).isEqualTo("new-hash");
        verify(refreshTokenService).revokeAllForUser(userId);
    }

    @Test
    void resetPassword_invalidToken_throws() {
        when(passwordResetTokenService.consume("bad")).thenReturn(Optional.empty());
        ResetPasswordRequest req = new ResetPasswordRequest();
        req.setToken("bad");
        req.setNewPassword("newPass");
        assertThatThrownBy(() -> authService.resetPassword(req))
                .isInstanceOf(InvalidTokenException.class);
    }
}
