package com.pizza.auth.user;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.Optional;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.orm.jpa.DataJpaTest;
import org.springframework.boot.test.autoconfigure.orm.jpa.TestEntityManager;

@DataJpaTest
class UserRepositoryTest {

    @Autowired
    private UserRepository userRepository;

    @Autowired
    private TestEntityManager entityManager;

    @Test
    void saveAndFindByEmail() {
        User user = new User("Test.User@Example.COM", "hashed-password", "Nguyen Van A");
        userRepository.save(user);
        entityManager.flush();
        entityManager.clear();

        Optional<User> found = userRepository.findByEmail("test.user@example.com");

        assertThat(found).isPresent();
        assertThat(found.get().getEmailAddress()).isEqualTo("test.user@example.com");
        assertThat(found.get().getFullName()).isEqualTo("Nguyen Van A");
        assertThat(found.get().getStatus()).isEqualTo(AccountStatus.PENDING_VERIFICATION);
        assertThat(found.get().getPasswordHash()).isEqualTo("hashed-password");
        assertThat(found.get().getCreatedAt()).isNotNull();
        assertThat(found.get().getUpdatedAt()).isNotNull();
    }

    @Test
    void existsByEmail() {
        User user = new User("exists@example.com", "hash", "Test User");
        userRepository.save(user);
        entityManager.flush();

        assertThat(userRepository.existsByEmail("exists@example.com")).isTrue();
        assertThat(userRepository.existsByEmail("not-exists@example.com")).isFalse();
    }
}
