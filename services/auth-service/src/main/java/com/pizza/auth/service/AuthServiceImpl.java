package com.pizza.auth.service;

import java.time.Instant;
import java.util.UUID;

import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

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
import io.jsonwebtoken.JwtException;

@Service
public class AuthServiceImpl implements AuthService {

    private final UserRepository userRepository;
    private final PasswordEncoder passwordEncoder;
    private final JwtService jwtService;
    private final RefreshTokenService refreshTokenService;
    private final TokenBlacklistService tokenBlacklistService;
    private final LoginAttemptService loginAttemptService;
    private final PasswordResetTokenService passwordResetTokenService;
    private final EmailService emailService;
    private final JwtProperties jwtProperties;
    private final SecurityProperties securityProperties;
    private final String dummyPasswordHash;

    public AuthServiceImpl(UserRepository userRepository,
            PasswordEncoder passwordEncoder,
            JwtService jwtService,
            RefreshTokenService refreshTokenService,
            TokenBlacklistService tokenBlacklistService,
            LoginAttemptService loginAttemptService,
            PasswordResetTokenService passwordResetTokenService,
            EmailService emailService,
            JwtProperties jwtProperties,
            SecurityProperties securityProperties) {
        this.userRepository = userRepository;
        this.passwordEncoder = passwordEncoder;
        this.jwtService = jwtService;
        this.refreshTokenService = refreshTokenService;
        this.tokenBlacklistService = tokenBlacklistService;
        this.loginAttemptService = loginAttemptService;
        this.passwordResetTokenService = passwordResetTokenService;
        this.emailService = emailService;
        this.jwtProperties = jwtProperties;
        this.securityProperties = securityProperties;
        this.dummyPasswordHash = passwordEncoder.encode("dummy-password-for-timing");
    }

    @Override
    @Transactional(noRollbackFor = InvalidCredentialsException.class)
    public TokenResponse login(LoginRequest request) {
        String email = request.getEmail().trim().toLowerCase();

        if (loginAttemptService.isLocked(email)) {
            throw new AccountLockedException();
        }

        User user = userRepository.findByEmail(email).orElse(null);
        if (user == null) {
            passwordEncoder.matches(request.getPassword(), dummyPasswordHash);
            loginAttemptService.recordFailure(email);
            throw new InvalidCredentialsException();
        }

        if (user.getLockedUntil() != null && user.getLockedUntil().isAfter(Instant.now())) {
            throw new AccountLockedException();
        }

        if (!passwordEncoder.matches(request.getPassword(), user.getPasswordHash())) {
            loginAttemptService.recordFailure(email);
            if (loginAttemptService.isLocked(email)) {
                user.setLockedUntil(Instant.now().plus(securityProperties.getLockDuration()));
                user.setStatus(AccountStatus.LOCKED);
                userRepository.save(user);
            }
            throw new InvalidCredentialsException();
        }

        if (user.getStatus() == AccountStatus.LOCKED) {
            if (user.getLockedUntil() != null && user.getLockedUntil().isAfter(Instant.now())) {
                throw new AccountLockedException();
            }
            user.setStatus(AccountStatus.ACTIVE);
            user.setLockedUntil(null);
            userRepository.save(user);
        }

        if (user.getStatus() != AccountStatus.ACTIVE) {
            throw new InvalidCredentialsException();
        }

        loginAttemptService.clearFailures(email);
        return issueTokens(user);
    }

    @Override
    public TokenResponse refresh(RefreshRequest request) {
        String refreshToken = request.getRefreshToken();
        try {
            Claims claims = jwtService.parseClaims(refreshToken);
            if (!jwtService.isRefreshToken(refreshToken)) {
                throw new InvalidTokenException("Not a refresh token");
            }
            String jti = claims.getId();
            UUID userId = UUID.fromString(claims.getSubject());
            if (tokenBlacklistService.isBlacklisted(jti)) {
                throw new InvalidTokenException("Token has been revoked");
            }
            if (!refreshTokenService.consume(userId, jti)) {
                refreshTokenService.revokeAllForUser(userId);
                throw new InvalidTokenException("Refresh token not found or expired");
            }
            User user = userRepository.findById(userId)
                    .orElseThrow(() -> new InvalidTokenException("User not found"));
            if (user.getStatus() != AccountStatus.ACTIVE) {
                throw new InvalidTokenException("Account is not active");
            }
            return issueTokens(user);
        } catch (JwtException ex) {
            throw new InvalidTokenException("Invalid or expired refresh token");
        }
    }

    @Override
    public void logout(String accessToken, String refreshToken) {
        try {
            if (accessToken != null && !accessToken.isBlank()) {
                String accessJti = jwtService.getJti(accessToken);
                tokenBlacklistService.blacklist(accessJti, jwtService.getExpiration(accessToken));
            }
            if (refreshToken != null && !refreshToken.isBlank()) {
                String refreshJti = jwtService.getJti(refreshToken);
                UUID userId = jwtService.getUserId(refreshToken);
                refreshTokenService.revoke(userId, refreshJti);
                tokenBlacklistService.blacklist(refreshJti, jwtService.getExpiration(refreshToken));
            }
        } catch (JwtException | IllegalArgumentException ex) {
        }
    }

    @Override
    @Transactional(readOnly = true)
    public void forgotPassword(ForgotPasswordRequest request) {
        String email = request.getEmail().trim().toLowerCase();
        User user = userRepository.findByEmail(email).orElse(null);
        if (user == null) {
            return;
        }
        String token = passwordResetTokenService.create(user.getId());
        emailService.sendPasswordResetEmail(user.getEmail(), token);
    }

    @Override
    @Transactional
    public void resetPassword(ResetPasswordRequest request) {
        UUID userId = passwordResetTokenService.consume(request.getToken())
                .orElseThrow(() -> new InvalidTokenException("Invalid or expired reset token"));
        User user = userRepository.findById(userId)
                .orElseThrow(() -> new InvalidTokenException("User not found"));
        user.setPasswordHash(passwordEncoder.encode(request.getNewPassword()));
        if (user.getStatus() == AccountStatus.LOCKED) {
            user.setStatus(AccountStatus.ACTIVE);
            user.setLockedUntil(null);
        }
        userRepository.save(user);
        refreshTokenService.revokeAllForUser(userId);
        loginAttemptService.clearFailures(user.getEmail());
    }

    private TokenResponse issueTokens(User user) {
        String accessJti = UUID.randomUUID().toString();
        String refreshJti = UUID.randomUUID().toString();
        String accessToken = jwtService.generateAccessToken(user, accessJti);
        String refreshToken = jwtService.generateRefreshToken(user, refreshJti);
        refreshTokenService.store(user.getId(), refreshJti, refreshToken);
        long expiresIn = jwtProperties.getAccessTokenExpiration().toSeconds();
        return new TokenResponse(accessToken, refreshToken, "Bearer", expiresIn);
    }
}
