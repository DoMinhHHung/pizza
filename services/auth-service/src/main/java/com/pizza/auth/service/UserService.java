package com.pizza.auth.service;

import com.pizza.auth.dto.RegisterRequest;
import com.pizza.auth.dto.RegisterResponse;

public interface UserService {

    RegisterResponse register(RegisterRequest request);
}
