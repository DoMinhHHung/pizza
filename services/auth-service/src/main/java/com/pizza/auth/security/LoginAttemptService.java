package com.pizza.auth.security;

import com.pizza.auth.config.SecurityProperties;
import java.time.Duration;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;

@Service
public class LoginAttemptService {

    private static final String PREFIX = "auth:fail:";

    private final StringRedisTemplate redisTemplate;
    private final int maxAttempts;
    private final Duration lockDuration;

    public LoginAttemptService(StringRedisTemplate redisTemplate, SecurityProperties securityProperties) {
        this.redisTemplate = redisTemplate;
        this.maxAttempts = securityProperties.getMaxFailedAttempts();
        this.lockDuration = securityProperties.getLockDuration();
    }

    public void recordFailure(String email) {
        String key = PREFIX + email;
        Long count = redisTemplate.opsForValue().increment(key);
        if (count != null && count == 1L) {
            redisTemplate.expire(key, lockDuration);
        }
    }

    public void clearFailures(String email) {
        redisTemplate.delete(PREFIX + email);
    }

    public boolean isLocked(String email) {
        String value = redisTemplate.opsForValue().get(PREFIX + email);
        if (value == null) {
            return false;
        }
        try {
            return Integer.parseInt(value) >= maxAttempts;
        } catch (NumberFormatException e) {
            return false;
        }
    }
}
