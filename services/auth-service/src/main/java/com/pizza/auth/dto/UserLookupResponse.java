package com.pizza.auth.dto;

import java.util.UUID;

import lombok.Getter;

@Getter
public class UserLookupResponse {
    private final UUID id;
    private final String email;
    private final String username;
    private final String fullName;

    public UserLookupResponse(UUID id, String email, String username, String fullName) {
        this.id = id;
        this.email = email;
        this.username = username;
        this.fullName = fullName;
    }
}
