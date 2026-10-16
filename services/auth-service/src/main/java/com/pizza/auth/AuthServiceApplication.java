package com.pizza.auth;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.EnableConfigurationProperties;

import com.pizza.auth.config.CorsProperties;
import com.pizza.auth.config.JwtProperties;
import com.pizza.auth.config.MailProperties;
import com.pizza.auth.config.RateLimitProperties;
import com.pizza.auth.config.SecurityProperties;

@SpringBootApplication
@EnableConfigurationProperties({
        JwtProperties.class,
        SecurityProperties.class,
        MailProperties.class,
        CorsProperties.class,
        RateLimitProperties.class
})
public class AuthServiceApplication {

    public static void main(String[] args) {
        SpringApplication.run(AuthServiceApplication.class, args);
    }
}
