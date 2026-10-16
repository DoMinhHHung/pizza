package com.pizza.auth.controller;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.doNothing;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.pizza.auth.dto.RegisterRequest;
import com.pizza.auth.user.UserRepository;
import java.time.Duration;
import java.util.concurrent.TimeUnit;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.Mockito;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.ValueOperations;
import org.springframework.http.MediaType;
import org.springframework.mail.javamail.JavaMailSender;
import org.springframework.test.web.servlet.MockMvc;

@SpringBootTest
@AutoConfigureMockMvc
class AuthControllerIntegrationTest {

    @MockBean
    private JavaMailSender javaMailSender;

    @MockBean
    private StringRedisTemplate stringRedisTemplate;

    @Autowired
    private MockMvc mockMvc;

    @Autowired
    private UserRepository userRepository;

    @Autowired
    private ObjectMapper objectMapper;

    @BeforeEach
    @SuppressWarnings("unchecked")
    void setUp() {
        userRepository.deleteAll();

        ValueOperations<String, String> ops = Mockito.mock(ValueOperations.class);
        when(stringRedisTemplate.opsForValue()).thenReturn(ops);
        when(ops.increment(anyString())).thenReturn(1L);
        doNothing().when(ops).set(anyString(), anyString());
        doNothing().when(ops).set(anyString(), anyString(), any(Duration.class));
        doNothing().when(ops).set(anyString(), anyString(), anyLong(), any(TimeUnit.class));
        when(ops.get(anyString())).thenReturn(null);
        when(ops.getAndDelete(anyString())).thenReturn(null);
        when(stringRedisTemplate.expire(anyString(), any(Duration.class))).thenReturn(true);
        when(stringRedisTemplate.expire(anyString(), anyLong(), any(TimeUnit.class))).thenReturn(true);
    }

    @Test
    void register_success() throws Exception {
        RegisterRequest request = new RegisterRequest();
        request.setEmail("new.user@example.com");
        request.setPassword("Str0ngP@ssw0rd!");
        request.setFullName("Nguyen Van A");

        mockMvc.perform(post("/api/v1/auth/register")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.id").exists())
                .andExpect(jsonPath("$.email").value("new.user@example.com"))
                .andExpect(jsonPath("$.fullName").value("Nguyen Van A"))
                .andExpect(jsonPath("$.status").value("PENDING_VERIFICATION"));
    }

    @Test
    void register_duplicateEmail_returns409() throws Exception {
        RegisterRequest request = new RegisterRequest();
        request.setEmail("duplicate@example.com");
        request.setPassword("Str0ngP@ssw0rd!");
        request.setFullName("First User");

        mockMvc.perform(post("/api/v1/auth/register")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isCreated());

        mockMvc.perform(post("/api/v1/auth/register")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isConflict())
                .andExpect(jsonPath("$.message").value("Email already registered"));
    }

    @Test
    void register_invalidPayload_returns400() throws Exception {
        RegisterRequest request = new RegisterRequest();
        request.setEmail("not-an-email");
        request.setPassword("short");
        request.setFullName("");

        mockMvc.perform(post("/api/v1/auth/register")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isBadRequest());
    }
}
