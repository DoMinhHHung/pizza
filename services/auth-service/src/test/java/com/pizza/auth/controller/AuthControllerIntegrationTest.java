package com.pizza.auth.controller;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.doNothing;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

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

import com.fasterxml.jackson.databind.ObjectMapper;
import com.pizza.auth.dto.RegisterRequest;
import com.pizza.auth.user.UserRepository;

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
        request.setUsername("new_user");
        request.setPassword("Str0ngP@ssw0rd!");
        request.setFullName("Nguyen Van A");

        mockMvc.perform(post("/api/v1/auth/register")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.id").exists())
                .andExpect(jsonPath("$.email").value("new.user@example.com"))
                .andExpect(jsonPath("$.username").value("new_user"))
                .andExpect(jsonPath("$.fullName").value("Nguyen Van A"))
                .andExpect(jsonPath("$.status").value("PENDING_VERIFICATION"));
    }

    @Test
    void register_duplicateEmail_returns409() throws Exception {
        RegisterRequest first = new RegisterRequest();
        first.setEmail("duplicate@example.com");
        first.setUsername("dup_user");
        first.setPassword("Str0ngP@ssw0rd!");
        first.setFullName("First User");

        mockMvc.perform(post("/api/v1/auth/register")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(first)))
                .andExpect(status().isCreated());

        // Cùng email, username khác → 409 vì email
        RegisterRequest second = new RegisterRequest();
        second.setEmail("duplicate@example.com");
        second.setUsername("dup_user_2");
        second.setPassword("Str0ngP@ssw0rd!");
        second.setFullName("Second User");

        mockMvc.perform(post("/api/v1/auth/register")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(second)))
                .andExpect(status().isConflict())
                .andExpect(jsonPath("$.message").value("Email already registered"));
    }

    @Test
    void register_invalidPayload_returns400() throws Exception {
        RegisterRequest request = new RegisterRequest();
        request.setEmail("not-an-email");
        request.setUsername("ab"); // quá ngắn nếu min=3
        request.setPassword("short");
        request.setFullName("");

        mockMvc.perform(post("/api/v1/auth/register")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isBadRequest());
    }
}
