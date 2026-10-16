package com.pizza.auth.security;

import com.pizza.auth.config.RateLimitProperties;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.time.Duration;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

@Component
public class RateLimitFilter extends OncePerRequestFilter {

    private static final String PREFIX = "rate:";

    private final StringRedisTemplate redisTemplate;
    private final RateLimitProperties properties;

    public RateLimitFilter(StringRedisTemplate redisTemplate, RateLimitProperties properties) {
        this.redisTemplate = redisTemplate;
        this.properties = properties;
    }

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response,
            FilterChain filterChain) throws ServletException, IOException {
        String path = request.getRequestURI();
        String method = request.getMethod();
        if (!"POST".equalsIgnoreCase(method)) {
            filterChain.doFilter(request, response);
            return;
        }
        String clientKey = request.getRemoteAddr();
        RateLimitRule rule = resolveRule(path);
        if (rule == null) {
            filterChain.doFilter(request, response);
            return;
        }
        String redisKey = PREFIX + rule.name() + ":" + clientKey;
        Long count = redisTemplate.opsForValue().increment(redisKey);
        if (count != null && count == 1L) {
            redisTemplate.expire(redisKey, rule.window());
        }
        if (count != null && count > rule.limit()) {
            response.setStatus(HttpStatus.TOO_MANY_REQUESTS.value());
            response.setContentType("application/json");
            response.getWriter().write("{\"message\":\"Too many requests. Please try again later.\"}");
            return;
        }
        filterChain.doFilter(request, response);
    }

    private RateLimitRule resolveRule(String path) {
        if (path.endsWith("/login")) {
            return new RateLimitRule("login", properties.getLoginRequests(), properties.getLoginWindow());
        }
        if (path.endsWith("/register")) {
            return new RateLimitRule("register", properties.getRegisterRequests(), properties.getRegisterWindow());
        }
        if (path.endsWith("/forgot-password") || path.endsWith("/resend-verification")) {
            return new RateLimitRule("forgot", properties.getForgotPasswordRequests(),
                    properties.getForgotPasswordWindow());
        }
        return null;
    }

    private record RateLimitRule(String name, int limit, Duration window) {
    }
}
