package com.pizza.auth.service;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doNothing;
import static org.mockito.Mockito.lenient;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.time.Duration;
import java.util.Optional;
import java.util.UUID;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.security.crypto.password.PasswordEncoder;

import com.pizza.auth.config.JwtProperties;
import com.pizza.auth.config.SecurityProperties;
import com.pizza.auth.dto.ForgotPasswordRequest;
import com.pizza.auth.dto.LoginRequest;
import com.pizza.auth.dto.RefreshRequest;
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

import io.jsonwebtoken.Claims;

@ExtendWith(MockitoExtension.class)
class AuthServiceTest {

    @Mock
    private UserRepository userRepository;
    @Mock
    private PasswordEncoder passwordEncoder;
    @Mock
    private JwtService jwtService;
    @Mock
    private RefreshTokenService refreshTokenService;
    @Mock
    private TokenBlacklistService tokenBlacklistService;
    @Mock
    private LoginAttemptService loginAttemptService;
    @Mock
    private PasswordResetTokenService passwordResetTokenService;
    @Mock
    private EmailService emailService;
    @Mock
    private JwtProperties jwtProperties;
    @Mock
    private SecurityProperties securityProperties;

    private AuthServiceImpl authService;
    private User activeUser;
    private LoginRequest loginRequest;

    @BeforeEach
    void setUp() {
        when(passwordEncoder.encode(anyString())).thenReturn("dummy-hash");
        lenient().when(jwtProperties.getAccessTokenExpiration()).thenReturn(Duration.ofMinutes(30));
        lenient().when(jwtProperties.getRefreshTokenExpiration()).thenReturn(Duration.ofDays(7));

        authService = new AuthServiceImpl(
                userRepository,
                passwordEncoder,
                jwtService,
                refreshTokenService,
                tokenBlacklistService,
                loginAttemptService,
                passwordResetTokenService,
                emailService,
                jwtProperties,
                securityProperties);

        activeUser = new User("user@example.com", "existing_username", "hashed", "Test User");
        activeUser.setStatus(AccountStatus.ACTIVE);

        loginRequest = new LoginRequest();
        loginRequest.setIdentifier("user@example.com");
        loginRequest.setPassword("password");
    }

    @Test
    void login_success() {
        when(loginAttemptService.isLocked("user@example.com")).thenReturn(false);
        when(userRepository.findByEmailOrUsername("user@example.com")).thenReturn(Optional.of(activeUser));
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
    void login_byUsername_success() {
        LoginRequest byUsername = new LoginRequest();
        byUsername.setIdentifier("existing_username");
        byUsername.setPassword("password");

        when(loginAttemptService.isLocked("existing_username")).thenReturn(false);
        when(userRepository.findByEmailOrUsername("existing_username")).thenReturn(Optional.of(activeUser));
        when(passwordEncoder.matches("password", "hashed")).thenReturn(true);
        when(jwtService.generateAccessToken(any(), anyString())).thenReturn("access");
        when(jwtService.generateRefreshToken(any(), anyString())).thenReturn("refresh");
        doNothing().when(refreshTokenService).store(any(), anyString(), anyString());
        doNothing().when(loginAttemptService).clearFailures(anyString());

        TokenResponse response = authService.login(byUsername);

        assertThat(response.getAccessToken()).isEqualTo("access");
        verify(userRepository).findByEmailOrUsername("existing_username");
    }

    @Test
    void login_pendingVerification_throws() {
        activeUser.setStatus(AccountStatus.PENDING_VERIFICATION);
        when(loginAttemptService.isLocked("user@example.com")).thenReturn(false);
        when(userRepository.findByEmailOrUsername("user@example.com")).thenReturn(Optional.of(activeUser));
        when(passwordEncoder.matches("password", "hashed")).thenReturn(true);

        assertThatThrownBy(() -> authService.login(loginRequest))
                .isInstanceOf(InvalidCredentialsException.class);
        verify(jwtService, never()).generateAccessToken(any(), anyString());
    }

    @Test
    void login_unknownIdentifier_recordsFailure() {
        when(loginAttemptService.isLocked("user@example.com")).thenReturn(false);
        when(userRepository.findByEmailOrUsername("user@example.com")).thenReturn(Optional.empty());
        when(passwordEncoder.matches(eq("password"), anyString())).thenReturn(false);

        assertThatThrownBy(() -> authService.login(loginRequest))
                .isInstanceOf(InvalidCredentialsException.class);
        verify(loginAttemptService).recordFailure("user@example.com");
    }

    @Test
    void login_wrongPassword_recordsFailure() {
        when(loginAttemptService.isLocked("user@example.com")).thenReturn(false);
        when(userRepository.findByEmailOrUsername("user@example.com")).thenReturn(Optional.of(activeUser));
        when(passwordEncoder.matches("password", "hashed")).thenReturn(false);

        assertThatThrownBy(() -> authService.login(loginRequest))
                .isInstanceOf(InvalidCredentialsException.class);
        verify(loginAttemptService).recordFailure("user@example.com");
    }

    @Test
    void login_lockedByAttempts_throws() {
        when(loginAttemptService.isLocked("user@example.com")).thenReturn(true);
        assertThatThrownBy(() -> authService.login(loginRequest))
                .isInstanceOf(AccountLockedException.class);
        verify(userRepository, never()).findByEmailOrUsername(anyString());
    }

    @Test
    void forgotPassword_existingUser_sendsEmail() {
        when(userRepository.findByEmailOrUsername("user@example.com")).thenReturn(Optional.of(activeUser));
        when(passwordResetTokenService.create(any())).thenReturn("reset-token");
        doNothing().when(emailService).sendPasswordResetEmail(anyString(), anyString());

        ForgotPasswordRequest req = new ForgotPasswordRequest();
        req.setIdentifier("user@example.com");
        authService.forgotPassword(req);

        verify(emailService).sendPasswordResetEmail(eq("user@example.com"), eq("reset-token"));
    }

    @Test
    void forgotPassword_unknownIdentifier_noOp() {
        when(userRepository.findByEmailOrUsername("unknown@example.com")).thenReturn(Optional.empty());
        ForgotPasswordRequest req = new ForgotPasswordRequest();
        req.setIdentifier("unknown@example.com");
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

    @Test
    void refresh_success_rotatesWithoutBlacklistingOrRevokingAll() {
        UUID userId = UUID.randomUUID();
        Claims claims = mock(Claims.class);
        when(claims.getId()).thenReturn("old-jti");
        when(claims.getSubject()).thenReturn(userId.toString());
        when(jwtService.parseClaims("old-refresh")).thenReturn(claims);
        when(jwtService.isRefreshToken("old-refresh")).thenReturn(true);
        when(tokenBlacklistService.isBlacklisted("old-jti")).thenReturn(false);
        when(refreshTokenService.consume(userId, "old-jti")).thenReturn(true);
        when(userRepository.findById(userId)).thenReturn(Optional.of(activeUser));
        when(jwtService.generateAccessToken(any(), anyString())).thenReturn("new-access");
        when(jwtService.generateRefreshToken(any(), anyString())).thenReturn("new-refresh");

        TokenResponse response = authService.refresh(refreshRequest("old-refresh"));

        assertThat(response.getAccessToken()).isEqualTo("new-access");
        assertThat(response.getRefreshToken()).isEqualTo("new-refresh");
        verify(refreshTokenService, never()).revokeAllForUser(any());
        verify(tokenBlacklistService, never()).blacklist(anyString(), any());
    }

    @Test
    void refresh_reusedToken_revokesAllSessionsAndThrows() {
        UUID userId = UUID.randomUUID();
        Claims claims = mock(Claims.class);
        when(claims.getId()).thenReturn("used-jti");
        when(claims.getSubject()).thenReturn(userId.toString());
        when(jwtService.parseClaims("used-refresh")).thenReturn(claims);
        when(jwtService.isRefreshToken("used-refresh")).thenReturn(true);
        when(tokenBlacklistService.isBlacklisted("used-jti")).thenReturn(false);
        when(refreshTokenService.consume(userId, "used-jti")).thenReturn(false);

        assertThatThrownBy(() -> authService.refresh(refreshRequest("used-refresh")))
                .isInstanceOf(InvalidTokenException.class);

        verify(refreshTokenService).revokeAllForUser(userId);
        verify(userRepository, never()).findById(any());
    }

    @Test
    void refresh_blacklistedToken_throwsWithoutConsuming() {
        UUID userId = UUID.randomUUID();
        Claims claims = mock(Claims.class);
        when(claims.getId()).thenReturn("logged-out-jti");
        when(claims.getSubject()).thenReturn(userId.toString());
        when(jwtService.parseClaims("logged-out-refresh")).thenReturn(claims);
        when(jwtService.isRefreshToken("logged-out-refresh")).thenReturn(true);
        when(tokenBlacklistService.isBlacklisted("logged-out-jti")).thenReturn(true);

        assertThatThrownBy(() -> authService.refresh(refreshRequest("logged-out-refresh")))
                .isInstanceOf(InvalidTokenException.class);

        verify(refreshTokenService, never()).consume(any(), anyString());
        verify(refreshTokenService, never()).revokeAllForUser(any());
    }

    @Test
    void refresh_accessTokenPresented_throws() {
        Claims claims = mock(Claims.class);
        when(jwtService.parseClaims("access-token")).thenReturn(claims);
        when(jwtService.isRefreshToken("access-token")).thenReturn(false);

        assertThatThrownBy(() -> authService.refresh(refreshRequest("access-token")))
                .isInstanceOf(InvalidTokenException.class);

        verify(refreshTokenService, never()).consume(any(), anyString());
    }

    private RefreshRequest refreshRequest(String token) {
        RefreshRequest request = new RefreshRequest();
        request.setRefreshToken(token);
        return request;
    }

    @Test
    void logout_expiredAccess_stillRevokesRefresh() {
        UUID userId = UUID.randomUUID();
        when(jwtService.getJti("expired-access"))
                .thenThrow(new io.jsonwebtoken.ExpiredJwtException(null, null, "expired"));
        when(jwtService.getJti("valid-refresh")).thenReturn("refresh-jti");
        when(jwtService.getUserId("valid-refresh")).thenReturn(userId);
        when(jwtService.getExpiration("valid-refresh")).thenReturn(java.time.Instant.now().plusSeconds(3600));
        doNothing().when(refreshTokenService).revoke(userId, "refresh-jti");
        doNothing().when(tokenBlacklistService).blacklist(eq("refresh-jti"), any());

        authService.logout("expired-access", "valid-refresh");

        verify(refreshTokenService).revoke(userId, "refresh-jti");
        verify(tokenBlacklistService).blacklist(eq("refresh-jti"), any());
        verify(tokenBlacklistService, never()).blacklist(eq("access-jti"), any());
    }

    @Test
    void logout_validAccessAndRefresh_blacklistsBoth() {
        UUID userId = UUID.randomUUID();
        when(jwtService.getJti("access")).thenReturn("access-jti");
        when(jwtService.getExpiration("access")).thenReturn(java.time.Instant.now().plusSeconds(900));
        when(jwtService.getJti("refresh")).thenReturn("refresh-jti");
        when(jwtService.getUserId("refresh")).thenReturn(userId);
        when(jwtService.getExpiration("refresh")).thenReturn(java.time.Instant.now().plusSeconds(3600));
        doNothing().when(tokenBlacklistService).blacklist(anyString(), any());
        doNothing().when(refreshTokenService).revoke(userId, "refresh-jti");

        authService.logout("access", "refresh");

        verify(tokenBlacklistService).blacklist(eq("access-jti"), any());
        verify(refreshTokenService).revoke(userId, "refresh-jti");
        verify(tokenBlacklistService).blacklist(eq("refresh-jti"), any());
    }

    @Test
    void logout_nullTokens_noOp() {
        authService.logout(null, null);
        verify(refreshTokenService, never()).revoke(any(), anyString());
        verify(tokenBlacklistService, never()).blacklist(anyString(), any());
    }
}
