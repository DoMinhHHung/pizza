package com.pizza.auth.dto;

import lombok.Getter;

@Getter
public class TokenResponse {

    private final String accessToken;
    private final String refreshToken;
    private final String tokenType;
    private final long expiresIn;

    public TokenResponse(String accessToken, String refreshToken, String tokenType, long expiresIn) {
        this.accessToken = accessToken;
        this.refreshToken = refreshToken;
        this.tokenType = tokenType;
        this.expiresIn = expiresIn;
    }
}
