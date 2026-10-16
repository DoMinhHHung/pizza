package com.pizza.auth.security;

import com.pizza.auth.config.SecurityProperties;
import java.time.Duration;
import java.util.Optional;
import java.util.UUID;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;

@Service
public class PasswordResetTokenService {

    private static final String PREFIX = "reset:password:";

    private final StringRedisTemplate redisTemplate;
    private final Duration ttl;

    public PasswordResetTokenService(StringRedisTemplate redisTemplate, SecurityProperties securityProperties) {
        this.redisTemplate = redisTemplate;
        this.ttl = securityProperties.getResetTokenTtl();
    }

    public String create(UUID userId) {
        String token = UUID.randomUUID().toString();
        redisTemplate.opsForValue().set(PREFIX + token, userId.toString(), ttl);
        return token;
    }

    public Optional<UUID> consume(String token) {
        String value = redisTemplate.opsForValue().getAndDelete(PREFIX + token);
        if (value == null) {
            return Optional.empty();
        }
        return Optional.of(UUID.fromString(value));
    }
}
