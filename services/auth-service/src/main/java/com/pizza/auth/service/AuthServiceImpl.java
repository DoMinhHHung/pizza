package com.pizza.auth.service;

import java.util.UUID;

import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import com.pizza.auth.dto.LoginRequest;
import com.pizza.auth.dto.RefreshRequest;
import com.pizza.auth.dto.TokenResponse;
import com.pizza.auth.exception.InvalidCredentialsException;
import com.pizza.auth.exception.InvalidTokenException;
import com.pizza.auth.security.JwtService;
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

    public AuthServiceImpl(UserRepository userRepository,
            PasswordEncoder passwordEncoder,
            JwtService jwtService,
            RefreshTokenService refreshTokenService,
            TokenBlacklistService tokenBlacklistService) {
        this.userRepository = userRepository;
        this.passwordEncoder = passwordEncoder;
        this.jwtService = jwtService;
        this.refreshTokenService = refreshTokenService;
        this.tokenBlacklistService = tokenBlacklistService;
    }

    @Override
    @Transactional(readOnly = true)
    public TokenResponse login(LoginRequest request) {
        String email = request.getEmail().trim().toLowerCase();

        User user = userRepository.findByEmail(email)
                .orElseThrow(InvalidCredentialsException::new);

        if (!passwordEncoder.matches(request.getPassword(), user.getPasswordHash())) {
            throw new InvalidCredentialsException();
        }

        if (user.getStatus() != AccountStatus.ACTIVE
                && user.getStatus() != AccountStatus.PENDING_VERIFICATION) {
            throw new InvalidCredentialsException();
        }

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

            if (!refreshTokenService.exists(userId, jti)) {
                throw new InvalidTokenException("Refresh token not found or expired");
            }

            User user = userRepository.findById(userId)
                    .orElseThrow(() -> new InvalidTokenException("User not found"));

            refreshTokenService.revoke(userId, jti);
            tokenBlacklistService.blacklist(jti, claims.getExpiration().toInstant());

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

    private TokenResponse issueTokens(User user) {
        String accessJti = UUID.randomUUID().toString();
        String refreshJti = UUID.randomUUID().toString();

        String accessToken = jwtService.generateAccessToken(user, accessJti);
        String refreshToken = jwtService.generateRefreshToken(user, refreshJti);

        refreshTokenService.store(user.getId(), refreshJti, refreshToken);

        long expiresIn = 30 * 60;

        return new TokenResponse(accessToken, refreshToken, "Bearer", expiresIn);
    }
}
