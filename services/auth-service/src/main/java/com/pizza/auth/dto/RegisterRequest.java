package com.pizza.auth.dto;

import jakarta.validation.constraints.Email;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Size;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
public class RegisterRequest {

    @NotBlank
    @Email
    @Size(max = 254)
    private String email;

    @NotBlank
    @Size(min = 8, max = 72)
    private String password;

    @NotBlank
    @Size(max = 100)
    private String fullName;

    @NotBlank
    @Size(min = 3, max = 30)
    @Pattern(regexp = "^[a-zA-Z0-9_]+$", message = "username must be alphanumeric or underscore")
    private String username;
}
