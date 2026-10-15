package com.pizza.auth.security;

import java.time.Duration;
import java.util.UUID;

import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;

@Service
public class RefreshTokenService {

    private static final String REFRESH_PREFIX = "refresh:";

    private final StringRedisTemplate redisTemplate;
    private final Duration refreshTtl;

    public RefreshTokenService(StringRedisTemplate redisTemplate,
            com.pizza.auth.config.JwtProperties jwtProperties) {
        this.redisTemplate = redisTemplate;
        this.refreshTtl = jwtProperties.getRefreshTokenExpiration();
    }

    public void store(UUID userId, String jti, String refreshToken) {
        String key = buildKey(userId, jti);
        redisTemplate.opsForValue().set(key, refreshToken, refreshTtl);
    }

    public boolean exists(UUID userId, String jti) {
        return Boolean.TRUE.equals(redisTemplate.hasKey(buildKey(userId, jti)));
    }

    public void revoke(UUID userId, String jti) {
        redisTemplate.delete(buildKey(userId, jti));
    }

    public void revokeAllForUser(UUID userId) {
        // không scan toàn bộ
        // Có thể mở rộng sau bằng Set. Hiện tại chỉ revoke từng jti
    }

    private String buildKey(UUID userId, String jti) {
        return REFRESH_PREFIX + userId + ":" + jti;
    }
}
