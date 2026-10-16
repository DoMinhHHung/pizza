package com.pizza.auth.service;

import com.pizza.auth.dto.ForgotPasswordRequest;
import com.pizza.auth.dto.LoginRequest;
import com.pizza.auth.dto.RefreshRequest;
import com.pizza.auth.dto.ResetPasswordRequest;
import com.pizza.auth.dto.TokenResponse;

public interface AuthService {

    TokenResponse login(LoginRequest request);

    TokenResponse refresh(RefreshRequest request);

    void logout(String accessToken, String refreshToken);

    void forgotPassword(ForgotPasswordRequest request);

    void resetPassword(ResetPasswordRequest request);
}
