package com.pizza.auth.service;

import com.pizza.auth.dto.ChangePasswordRequest;
import com.pizza.auth.dto.RegisterRequest;
import com.pizza.auth.dto.RegisterResponse;
import com.pizza.auth.dto.UserResponse;
import com.pizza.auth.exception.EmailAlreadyExistsException;
import com.pizza.auth.exception.InvalidCurrentPasswordException;
import com.pizza.auth.exception.InvalidTokenException;
import com.pizza.auth.security.RefreshTokenService;
import com.pizza.auth.security.VerificationTokenService;
import com.pizza.auth.user.AccountStatus;
import com.pizza.auth.user.User;
import com.pizza.auth.user.UserRepository;
import java.time.Instant;
import java.util.UUID;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class UserServiceImpl implements UserService {

    private static final Logger log = LoggerFactory.getLogger(UserServiceImpl.class);

    private final UserRepository userRepository;
    private final PasswordEncoder passwordEncoder;
    private final VerificationTokenService verificationTokenService;
    private final EmailService emailService;
    private final RefreshTokenService refreshTokenService;

    public UserServiceImpl(UserRepository userRepository,
            PasswordEncoder passwordEncoder,
            VerificationTokenService verificationTokenService,
            EmailService emailService,
            RefreshTokenService refreshTokenService) {
        this.userRepository = userRepository;
        this.passwordEncoder = passwordEncoder;
        this.verificationTokenService = verificationTokenService;
        this.emailService = emailService;
        this.refreshTokenService = refreshTokenService;
    }

    @Override
    @Transactional
    public RegisterResponse register(RegisterRequest request) {
        String normalizedEmail = request.getEmail().trim().toLowerCase();
        if (userRepository.existsByEmail(normalizedEmail)) {
            throw new EmailAlreadyExistsException(normalizedEmail);
        }
        String passwordHash = passwordEncoder.encode(request.getPassword());
        User user = new User(normalizedEmail, passwordHash, request.getFullName().trim());
        try {
            User saved = userRepository.saveAndFlush(user);
            String token = verificationTokenService.create(saved.getId());
            try {
                emailService.sendVerificationEmail(saved.getEmail(), token);
            } catch (Exception ex) {
                log.error("Failed to send verification email to {}", saved.getEmail(), ex);
            }
            return new RegisterResponse(
                    saved.getId(),
                    saved.getEmail(),
                    saved.getFullName(),
                    saved.getStatus());
        } catch (DataIntegrityViolationException ex) {
            throw new EmailAlreadyExistsException(normalizedEmail);
        }
    }

    @Override
    @Transactional
    public void verifyEmail(String token) {
        UUID userId = verificationTokenService.consume(token)
                .orElseThrow(() -> new InvalidTokenException("Invalid or expired verification token"));
        User user = userRepository.findById(userId)
                .orElseThrow(() -> new InvalidTokenException("User not found"));
        if (user.getStatus() != AccountStatus.PENDING_VERIFICATION) {
            return;
        }
        user.setStatus(AccountStatus.ACTIVE);
        user.setEmailVerifiedAt(Instant.now());
        userRepository.save(user);
    }

    @Override
    @Transactional(readOnly = true)
    public void resendVerification(String email) {
        String normalized = email.trim().toLowerCase();
        User user = userRepository.findByEmail(normalized).orElse(null);
        if (user == null || user.getStatus() != AccountStatus.PENDING_VERIFICATION) {
            return;
        }
        String token = verificationTokenService.create(user.getId());
        try {
            emailService.sendVerificationEmail(user.getEmail(), token);
        } catch (Exception ex) {
            log.error("Failed to resend verification email to {}", user.getEmail(), ex);
        }
    }

    @Override
    @Transactional(readOnly = true)
    public UserResponse getCurrentUser(UUID userId) {
        User user = userRepository.findById(userId)
                .orElseThrow(() -> new InvalidTokenException("User not found"));
        return new UserResponse(
                user.getId(),
                user.getEmail(),
                user.getFullName(),
                user.getStatus(),
                user.getEmailVerifiedAt(),
                user.getCreatedAt());
    }

    @Override
    @Transactional
    public void changePassword(UUID userId, ChangePasswordRequest request) {
        User user = userRepository.findById(userId)
                .orElseThrow(() -> new InvalidTokenException("User not found"));
        if (!passwordEncoder.matches(request.getCurrentPassword(), user.getPasswordHash())) {
            throw new InvalidCurrentPasswordException();
        }
        user.setPasswordHash(passwordEncoder.encode(request.getNewPassword()));
        userRepository.save(user);
        refreshTokenService.revokeAllForUser(userId);
    }
}
