package com.pizza.auth.dto;

import java.time.Instant;
import java.util.UUID;

import com.pizza.auth.user.AccountStatus;

import lombok.Getter;

@Getter
public class UserResponse {

    private final UUID id;
    private final String email;
    private final String username;
    private final String fullName;
    private final AccountStatus status;
    private final Instant emailVerifiedAt;
    private final Instant createdAt;

    public UserResponse(UUID id, String email, String username, String fullName,
            AccountStatus status, Instant emailVerifiedAt, Instant createdAt) {
        this.id = id;
        this.email = email;
        this.username = username;
        this.fullName = fullName;
        this.status = status;
        this.emailVerifiedAt = emailVerifiedAt;
        this.createdAt = createdAt;
    }
}
