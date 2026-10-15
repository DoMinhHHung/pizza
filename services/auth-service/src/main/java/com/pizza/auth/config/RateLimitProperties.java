package com.pizza.auth.config;

import java.time.Duration;
import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "app.security.rate-limit")
public class RateLimitProperties {

    private int loginRequests = 10;
    private Duration loginWindow = Duration.ofMinutes(1);
    private int registerRequests = 5;
    private Duration registerWindow = Duration.ofMinutes(1);
    private int forgotPasswordRequests = 3;
    private Duration forgotPasswordWindow = Duration.ofMinutes(1);

    public int getLoginRequests() {
        return loginRequests;
    }

    public void setLoginRequests(int loginRequests) {
        this.loginRequests = loginRequests;
    }

    public Duration getLoginWindow() {
        return loginWindow;
    }

    public void setLoginWindow(Duration loginWindow) {
        this.loginWindow = loginWindow;
    }

    public int getRegisterRequests() {
        return registerRequests;
    }

    public void setRegisterRequests(int registerRequests) {
        this.registerRequests = registerRequests;
    }

    public Duration getRegisterWindow() {
        return registerWindow;
    }

    public void setRegisterWindow(Duration registerWindow) {
        this.registerWindow = registerWindow;
    }

    public int getForgotPasswordRequests() {
        return forgotPasswordRequests;
    }

    public void setForgotPasswordRequests(int forgotPasswordRequests) {
        this.forgotPasswordRequests = forgotPasswordRequests;
    }

    public Duration getForgotPasswordWindow() {
        return forgotPasswordWindow;
    }

    public void setForgotPasswordWindow(Duration forgotPasswordWindow) {
        this.forgotPasswordWindow = forgotPasswordWindow;
    }
}
