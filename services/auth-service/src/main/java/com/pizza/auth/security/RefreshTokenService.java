package com.pizza.auth.security;

import com.pizza.auth.config.JwtProperties;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Duration;
import java.util.HexFormat;
import java.util.Set;
import java.util.UUID;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;

@Service
public class RefreshTokenService {

    private static final String TOKEN_PREFIX = "refresh:";
    private static final String USER_SET_PREFIX = "refresh:user:";

    private final StringRedisTemplate redisTemplate;
    private final Duration refreshTtl;

    public RefreshTokenService(StringRedisTemplate redisTemplate, JwtProperties jwtProperties) {
        this.redisTemplate = redisTemplate;
        this.refreshTtl = jwtProperties.getRefreshTokenExpiration();
    }

    public void store(UUID userId, String jti, String refreshToken) {
        String tokenKey = buildTokenKey(userId, jti);
        String setKey = buildUserSetKey(userId);
        redisTemplate.opsForValue().set(tokenKey, sha256Hex(refreshToken), refreshTtl);
        redisTemplate.opsForSet().add(setKey, jti);
        redisTemplate.expire(setKey, refreshTtl);
    }

    public boolean consume(UUID userId, String jti) {
        boolean removed = Boolean.TRUE.equals(redisTemplate.delete(buildTokenKey(userId, jti)));
        redisTemplate.opsForSet().remove(buildUserSetKey(userId), jti);
        return removed;
    }

    public boolean exists(UUID userId, String jti) {
        return Boolean.TRUE.equals(redisTemplate.hasKey(buildTokenKey(userId, jti)));
    }

    public void revoke(UUID userId, String jti) {
        redisTemplate.delete(buildTokenKey(userId, jti));
        redisTemplate.opsForSet().remove(buildUserSetKey(userId), jti);
    }

    public void revokeAllForUser(UUID userId) {
        String setKey = buildUserSetKey(userId);
        Set<String> jtis = redisTemplate.opsForSet().members(setKey);
        if (jtis != null && !jtis.isEmpty()) {
            for (String jti : jtis) {
                redisTemplate.delete(buildTokenKey(userId, jti));
            }
        }
        redisTemplate.delete(setKey);
    }

    private String buildTokenKey(UUID userId, String jti) {
        return TOKEN_PREFIX + userId + ":" + jti;
    }

    private String buildUserSetKey(UUID userId) {
        return USER_SET_PREFIX + userId;
    }

    private static String sha256Hex(String value) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            return HexFormat.of().formatHex(digest.digest(value.getBytes(StandardCharsets.UTF_8)));
        } catch (NoSuchAlgorithmException ex) {
            throw new IllegalStateException("SHA-256 is not available", ex);
        }
    }
}
