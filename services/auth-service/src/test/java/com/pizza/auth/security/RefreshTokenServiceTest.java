package com.pizza.auth.security;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import com.pizza.auth.config.JwtProperties;
import java.time.Duration;
import java.util.UUID;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.data.redis.core.SetOperations;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.ValueOperations;

@ExtendWith(MockitoExtension.class)
class RefreshTokenServiceTest {

    private static final UUID USER_ID = UUID.fromString("11111111-1111-1111-1111-111111111111");
    private static final String TOKEN_KEY = "refresh:" + USER_ID + ":jti-1";
    private static final String USER_SET_KEY = "refresh:user:" + USER_ID;

    @Mock
    private StringRedisTemplate redisTemplate;

    @Mock
    private ValueOperations<String, String> valueOperations;

    @Mock
    private SetOperations<String, String> setOperations;

    private RefreshTokenService service;

    @BeforeEach
    void setUp() {
        service = new RefreshTokenService(redisTemplate, new JwtProperties());
    }

    @Test
    void store_savesHashInsteadOfRawToken() {
        when(redisTemplate.opsForValue()).thenReturn(valueOperations);
        when(redisTemplate.opsForSet()).thenReturn(setOperations);

        service.store(USER_ID, "jti-1", "raw.refresh.token");

        ArgumentCaptor<String> stored = ArgumentCaptor.forClass(String.class);
        verify(valueOperations).set(eq(TOKEN_KEY), stored.capture(), eq(Duration.ofDays(14)));
        assertThat(stored.getValue()).isNotEqualTo("raw.refresh.token").hasSize(64).matches("[0-9a-f]{64}");
        verify(setOperations).add(USER_SET_KEY, "jti-1");
    }

    @Test
    void consume_returnsTrueWhenKeyWasRemoved() {
        when(redisTemplate.delete(TOKEN_KEY)).thenReturn(true);
        when(redisTemplate.opsForSet()).thenReturn(setOperations);

        assertThat(service.consume(USER_ID, "jti-1")).isTrue();

        verify(setOperations).remove(USER_SET_KEY, "jti-1");
    }

    @Test
    void consume_returnsFalseWhenKeyAlreadyGone() {
        when(redisTemplate.delete(TOKEN_KEY)).thenReturn(false);
        when(redisTemplate.opsForSet()).thenReturn(setOperations);

        assertThat(service.consume(USER_ID, "jti-1")).isFalse();
    }
}
