package com.pizza.auth.dto;

import com.pizza.auth.user.AccountStatus;
import java.util.UUID;
import lombok.AllArgsConstructor;
import lombok.Getter;

@Getter
@AllArgsConstructor
public class RegisterResponse {

    private final UUID id;
    private final String email;
    private final String fullName;
    private final AccountStatus status;
}
