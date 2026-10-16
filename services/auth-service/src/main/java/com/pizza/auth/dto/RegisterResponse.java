package com.pizza.auth.dto;

import java.util.UUID;
import com.pizza.auth.user.AccountStatus;
import lombok.Getter;

@Getter
public class RegisterResponse {

    private final UUID id;
    private final String email;
    private final String fullName;
    private final AccountStatus status;

    public RegisterResponse(UUID id, String email, String fullName, AccountStatus status) {
        this.id = id;
        this.email = email;
        this.fullName = fullName;
        this.status = status;
    }
}
