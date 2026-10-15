package com.pizza.auth.service;

import com.pizza.auth.config.MailProperties;
import org.springframework.mail.SimpleMailMessage;
import org.springframework.mail.javamail.JavaMailSender;
import org.springframework.stereotype.Service;

@Service
public class EmailServiceImpl implements EmailService {

    private final JavaMailSender mailSender;
    private final MailProperties mailProperties;

    public EmailServiceImpl(JavaMailSender mailSender, MailProperties mailProperties) {
        this.mailSender = mailSender;
        this.mailProperties = mailProperties;
    }

    @Override
    public void sendVerificationEmail(String to, String token) {
        String verifyUrl = mailProperties.getBaseUrl() + "/api/v1/auth/verify-email?token=" + token;
        SimpleMailMessage message = new SimpleMailMessage();
        message.setFrom(mailProperties.getFrom());
        message.setTo(to);
        message.setSubject("Verify your Pizza account");
        message.setText(
                "Welcome to Pizza.\n\n"
                        + "Please verify your email by clicking the link below or using the token:\n\n"
                        + verifyUrl + "\n\n"
                        + "Token: " + token + "\n\n"
                        + "This link expires in 24 hours.\n"
                        + "If you did not create an account, ignore this email.");
        mailSender.send(message);
    }

    @Override
    public void sendPasswordResetEmail(String to, String token) {
        String resetUrl = mailProperties.getBaseUrl() + "/api/v1/auth/reset-password?token=" + token;
        SimpleMailMessage message = new SimpleMailMessage();
        message.setFrom(mailProperties.getFrom());
        message.setTo(to);
        message.setSubject("Reset your Pizza password");
        message.setText(
                "You requested a password reset.\n\n"
                        + "Use the link below or the token to reset your password:\n\n"
                        + resetUrl + "\n\n"
                        + "Token: " + token + "\n\n"
                        + "This link expires in 1 hour.\n"
                        + "If you did not request this, ignore this email.");
        mailSender.send(message);
    }
}
