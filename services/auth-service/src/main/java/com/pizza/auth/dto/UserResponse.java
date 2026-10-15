package com.pizza.auth.dto;

import com.pizza.auth.user.AccountStatus;
import java.time.Instant;
import java.util.UUID;
import lombok.Getter;

@Getter
public class UserResponse {

    private final UUID id;
    private final String email;
    private final String fullName;
    private final AccountStatus status;
    private final Instant emailVerifiedAt;
    private final Instant createdAt;

    public UserResponse(UUID id, String email, String fullName, AccountStatus status,
            Instant emailVerifiedAt, Instant createdAt) {
        this.id = id;
        this.email = email;
        this.fullName = fullName;
        this.status = status;
        this.emailVerifiedAt = emailVerifiedAt;
        this.createdAt = createdAt;
    }
}
