package com.pizza.auth.controller;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.util.UUID;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.mail.javamail.JavaMailSender;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;
import org.testcontainers.containers.PostgreSQLContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.pizza.auth.dto.ChangePasswordRequest;
import com.pizza.auth.dto.LoginRequest;
import com.pizza.auth.dto.RegisterRequest;
import com.pizza.auth.dto.VerifyEmailRequest;
import com.pizza.auth.security.VerificationTokenService;
import com.pizza.auth.user.AccountStatus;
import com.pizza.auth.user.User;
import com.pizza.auth.user.UserRepository;
import com.redis.testcontainers.RedisContainer;

@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
@AutoConfigureMockMvc
@Testcontainers
class AuthServiceIT {

        @Container
        static PostgreSQLContainer<?> postgres = new PostgreSQLContainer<>(DockerImageName.parse("postgres:16-alpine"))
                        .withDatabaseName("pizza_auth")
                        .withUsername("pizza")
                        .withPassword("pizza");

        @Container
        static RedisContainer redis = new RedisContainer(DockerImageName.parse("redis:7-alpine"));

        @DynamicPropertySource
        static void registerProperties(DynamicPropertyRegistry registry) {
                registry.add("spring.datasource.url", postgres::getJdbcUrl);
                registry.add("spring.datasource.username", postgres::getUsername);
                registry.add("spring.datasource.password", postgres::getPassword);
                registry.add("spring.datasource.driver-class-name", postgres::getDriverClassName);
                registry.add("spring.jpa.properties.hibernate.dialect",
                                () -> "org.hibernate.dialect.PostgreSQLDialect");
                registry.add("spring.data.redis.url",
                                () -> "redis://" + redis.getHost() + ":" + redis.getFirstMappedPort());
                registry.add("app.jwt.secret", () -> "test-jwt-secret-key-at-least-32-characters-long");
                registry.add("app.mail.from", () -> "noreply@test.local");
                registry.add("app.mail.base-url", () -> "http://localhost:8080");
                registry.add("management.health.mail.enabled", () -> "false");
                registry.add("app.security.rate-limit.login-requests", () -> "100");
                registry.add("app.security.rate-limit.register-requests", () -> "100");
                registry.add("app.security.rate-limit.forgot-password-requests", () -> "100");
        }

        @Autowired
        private MockMvc mockMvc;

        @Autowired
        private ObjectMapper objectMapper;

        @Autowired
        private UserRepository userRepository;

        @Autowired
        private VerificationTokenService verificationTokenService;

        @MockBean
        private JavaMailSender javaMailSender;

        @BeforeEach
        void clean() {
                userRepository.deleteAll();
        }

        @Test
        void fullAuthFlow_register_verify_login_me_changePassword() throws Exception {
                RegisterRequest register = new RegisterRequest();
                register.setEmail("flow@example.com");
                register.setUsername("flow_user");
                register.setPassword("Str0ngP@ssw0rd!");
                register.setFullName("Flow User");

                MvcResult regResult = mockMvc.perform(post("/api/v1/auth/register")
                                .contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(register)))
                                .andExpect(status().isCreated())
                                .andExpect(jsonPath("$.status").value("PENDING_VERIFICATION"))
                                .andExpect(jsonPath("$.username").value("flow_user"))
                                .andReturn();

                JsonNode regBody = objectMapper.readTree(regResult.getResponse().getContentAsString());
                UUID userId = UUID.fromString(regBody.get("id").asText());

                String token = verificationTokenService.create(userId);
                VerifyEmailRequest verifyReq = new VerifyEmailRequest();
                verifyReq.setToken(token);

                mockMvc.perform(post("/api/v1/auth/verify-email")
                                .contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(verifyReq)))
                                .andExpect(status().isOk());

                User verified = userRepository.findById(userId).orElseThrow();
                assertThat(verified.getStatus()).isEqualTo(AccountStatus.ACTIVE);
                assertThat(verified.getEmailVerifiedAt()).isNotNull();
                assertThat(verified.getUsername()).isEqualTo("flow_user");

                // Login bằng email
                LoginRequest login = new LoginRequest();
                login.setIdentifier("flow@example.com");
                login.setPassword("Str0ngP@ssw0rd!");

                MvcResult loginResult = mockMvc.perform(post("/api/v1/auth/login")
                                .contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(login)))
                                .andExpect(status().isOk())
                                .andExpect(jsonPath("$.accessToken").exists())
                                .andExpect(jsonPath("$.refreshToken").exists())
                                .andReturn();

                JsonNode loginBody = objectMapper.readTree(loginResult.getResponse().getContentAsString());
                String accessToken = loginBody.get("accessToken").asText();

                mockMvc.perform(get("/api/v1/auth/me")
                                .header("Authorization", "Bearer " + accessToken))
                                .andExpect(status().isOk())
                                .andExpect(jsonPath("$.email").value("flow@example.com"))
                                .andExpect(jsonPath("$.username").value("flow_user"))
                                .andExpect(jsonPath("$.status").value("ACTIVE"));

                // Login bằng username
                LoginRequest loginByUsername = new LoginRequest();
                loginByUsername.setIdentifier("flow_user");
                loginByUsername.setPassword("Str0ngP@ssw0rd!");
                mockMvc.perform(post("/api/v1/auth/login")
                                .contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(loginByUsername)))
                                .andExpect(status().isOk())
                                .andExpect(jsonPath("$.accessToken").exists());

                ChangePasswordRequest change = new ChangePasswordRequest();
                change.setCurrentPassword("Str0ngP@ssw0rd!");
                change.setNewPassword("N3wStr0ngP@ss!");

                mockMvc.perform(post("/api/v1/auth/change-password")
                                .header("Authorization", "Bearer " + accessToken)
                                .contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(change)))
                                .andExpect(status().isOk());

                LoginRequest oldLogin = new LoginRequest();
                oldLogin.setIdentifier("flow@example.com");
                oldLogin.setPassword("Str0ngP@ssw0rd!");
                mockMvc.perform(post("/api/v1/auth/login")
                                .contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(oldLogin)))
                                .andExpect(status().isUnauthorized());

                LoginRequest newLogin = new LoginRequest();
                newLogin.setIdentifier("flow@example.com");
                newLogin.setPassword("N3wStr0ngP@ss!");
                mockMvc.perform(post("/api/v1/auth/login")
                                .contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(newLogin)))
                                .andExpect(status().isOk());
        }

        @Test
        void register_duplicateEmail_returns409() throws Exception {
                RegisterRequest first = new RegisterRequest();
                first.setEmail("dup@example.com");
                first.setUsername("dup_user");
                first.setPassword("Str0ngP@ssw0rd!");
                first.setFullName("Dup");

                mockMvc.perform(post("/api/v1/auth/register")
                                .contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(first)))
                                .andExpect(status().isCreated());

                RegisterRequest second = new RegisterRequest();
                second.setEmail("dup@example.com");
                second.setUsername("dup_user_2");
                second.setPassword("Str0ngP@ssw0rd!");
                second.setFullName("Dup 2");

                mockMvc.perform(post("/api/v1/auth/register")
                                .contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(second)))
                                .andExpect(status().isConflict())
                                .andExpect(jsonPath("$.message").value("Email already registered"));
        }

        @Test
        void login_wrongPassword_returns401() throws Exception {
                RegisterRequest register = new RegisterRequest();
                register.setEmail("wrong@example.com");
                register.setUsername("wrong_user");
                register.setPassword("Str0ngP@ssw0rd!");
                register.setFullName("Wrong");

                mockMvc.perform(post("/api/v1/auth/register")
                                .contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(register)))
                                .andExpect(status().isCreated());

                LoginRequest login = new LoginRequest();
                login.setIdentifier("wrong@example.com");
                login.setPassword("bad-password");

                mockMvc.perform(post("/api/v1/auth/login")
                                .contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(login)))
                                .andExpect(status().isUnauthorized());
        }

        @Test
        void me_withoutToken_returns401() throws Exception {
                mockMvc.perform(get("/api/v1/auth/me"))
                                .andExpect(status().isUnauthorized());
        }

        @Test
        void me_withInvalidToken_returns401WithBearerChallenge() throws Exception {
                mockMvc.perform(get("/api/v1/auth/me")
                                .header("Authorization", "Bearer not-a-valid-jwt"))
                                .andExpect(status().isUnauthorized())
                                .andExpect(header().string("WWW-Authenticate", "Bearer"))
                                .andExpect(jsonPath("$.message").value("Authentication required"));
        }
}
