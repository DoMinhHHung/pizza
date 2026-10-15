package com.pizza.auth.service;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import com.pizza.auth.dto.RegisterRequest;
import com.pizza.auth.dto.RegisterResponse;
import com.pizza.auth.exception.EmailAlreadyExistsException;
import com.pizza.auth.user.AccountStatus;
import com.pizza.auth.user.User;
import com.pizza.auth.user.UserRepository;
import java.util.UUID;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.security.crypto.password.PasswordEncoder;

@ExtendWith(MockitoExtension.class)
class UserServiceTest {

    @Mock
    private UserRepository userRepository;

    @Mock
    private PasswordEncoder passwordEncoder;

    @InjectMocks
    private UserService userService;

    private RegisterRequest request;

    @BeforeEach
    void setUp() {
        request = new RegisterRequest();
        request.setEmail("Test.User@Example.COM");
        request.setPassword("Str0ngP@ss!");
        request.setFullName("Nguyen Van A");
    }

    @Test
    void register_success() {
        when(userRepository.existsByEmail("test.user@example.com")).thenReturn(false);
        when(passwordEncoder.encode("Str0ngP@ss!")).thenReturn("hashed-password");

        User savedUser = new User("test.user@example.com", "hashed-password", "Nguyen Van A");
        when(userRepository.save(any(User.class))).thenAnswer(invocation -> {
            User u = invocation.getArgument(0);
            return u;
        });

        RegisterResponse response = userService.register(request);

        assertThat(response.getEmail()).isEqualTo("test.user@example.com");
        assertThat(response.getFullName()).isEqualTo("Nguyen Van A");
        assertThat(response.getStatus()).isEqualTo(AccountStatus.PENDING_VERIFICATION);

        ArgumentCaptor<User> captor = ArgumentCaptor.forClass(User.class);
        verify(userRepository).save(captor.capture());
        assertThat(captor.getValue().getPasswordHash()).isEqualTo("hashed-password");
    }

    @Test
    void register_emailAlreadyExists_throws() {
        when(userRepository.existsByEmail("test.user@example.com")).thenReturn(true);

        assertThatThrownBy(() -> userService.register(request))
                .isInstanceOf(EmailAlreadyExistsException.class);
    }
}
