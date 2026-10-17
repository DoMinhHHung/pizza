package com.pizza.auth.exception;

public class UsernameAlreadyExistsException extends RuntimeException {

    public UsernameAlreadyExistsException(String username) {
        super("Username already exists: " + username);
    }

}
