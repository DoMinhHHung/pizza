package com.pizza.auth.service;

import com.pizza.auth.dto.ChangePasswordRequest;
import com.pizza.auth.dto.RegisterRequest;
import com.pizza.auth.dto.RegisterResponse;
import com.pizza.auth.dto.UserResponse;
import java.util.UUID;

public interface UserService {

    RegisterResponse register(RegisterRequest request);

    void verifyEmail(String token);

    void resendVerification(String email);

    UserResponse getCurrentUser(UUID userId);

    void changePassword(UUID userId, ChangePasswordRequest request);
}
