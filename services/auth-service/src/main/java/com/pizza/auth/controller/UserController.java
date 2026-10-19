package com.pizza.auth.controller;

import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import com.pizza.auth.dto.UserLookupResponse;
import com.pizza.auth.service.UserService;

@RestController
@RequestMapping("/api/v1/users")
public class UserController {

    private final UserService userService;

    public UserController(UserService userService) {
        this.userService = userService;
    }

    @GetMapping("/lookup")
    public ResponseEntity<UserLookupResponse> lookup(
            @RequestParam("q") String q,
            Authentication authentication) {
        return ResponseEntity.ok(userService.lookup(q));
    }
}
