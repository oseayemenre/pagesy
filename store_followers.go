package main

import (
	"context"
	"errors"
	"fmt"
)

var (
	errUserNotFound    = errors.New("user not found")
	errUserNotFollowed = errors.New("user not followed")
)

func (s *server) checkIfUserExistsByID(ctx context.Context, userID string) error {
	var exists bool

	query :=
		`
			SELECT EXISTS(SELECT 1 FROM users WHERE id = $1);
		`

	if err := s.store.QueryRowContext(ctx, query, userID).Scan(&exists); err != nil {
		return fmt.Errorf("error checking if user exists, %v", err)
	}

	if !exists {
		return errUserNotFound
	}

	return nil
}

func (s *server) followUser(ctx context.Context, followerID, userID string) (string, error) {
	if err := s.checkIfUserExistsByID(ctx, userID); err != nil {
		return "", err
	}

	var displayName string

	query :=
		`
			SELECT display_name FROM users WHERE id = $1;
		`

	if err := s.store.QueryRowContext(ctx, query, followerID).Scan(&displayName); err != nil {
		return "", fmt.Errorf("error getting username, %v", err)
	}

	query =
		`
			INSERT INTO followers(user_id, follower_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;
		`

	results, err := s.store.ExecContext(ctx, query, userID, followerID)
	if err != nil {
		return "", fmt.Errorf("error inserting in followers table, %v", err)
	}

	rows, err := results.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("error checking number of rows affected, %v", err)
	}
	if rows > 0 {
		query :=
			`
				UPDATE users SET followers = followers + 1 WHERE id = $1;
			`

		if _, err := s.store.ExecContext(ctx, query, userID); err != nil {
			return "", fmt.Errorf("error updating user followers count, %v", err)
		}

		query =
			`
				UPDATE users SET following = following + 1 WHERE id = $1;
			`

		if _, err := s.store.ExecContext(ctx, query, followerID); err != nil {
			return "", fmt.Errorf("error updating user followers count, %v", err)
		}

		query =
			`
				INSERT INTO notifications (user_id, message) VALUES ($1, $2);
			`

		if _, err := s.store.ExecContext(ctx, query, userID, fmt.Sprintf("%v followed you", displayName)); err != nil {
			return "", fmt.Errorf("error inserting into notifications table, %v", err)
		}
	}

	return displayName, nil
}

func (s *server) unfollowUser(ctx context.Context, followerID, userID string) error {
	if err := s.checkIfUserExistsByID(ctx, userID); err != nil {
		return err
	}

	query :=
		`
			DELETE FROM followers WHERE user_id = $1 AND follower_id = $2;
		`

	results, err := s.store.ExecContext(ctx, query, userID, followerID)
	if err != nil {
		return fmt.Errorf("error deleting follower, %v", err)
	}

	rows, err := results.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking number of rows affected, %v", err)
	}
	if rows == 0 {
		return errUserNotFollowed
	}

	query =
		`
			UPDATE users SET followers = followers - 1 WHERE id = $1;
		`

	if _, err := s.store.ExecContext(ctx, query, userID); err != nil {
		return fmt.Errorf("error updating user followers count, %v", err)
	}

	query =
		`
			UPDATE users SET following = following - 1 WHERE id = $1;
		`

	if _, err := s.store.ExecContext(ctx, query, followerID); err != nil {
		return fmt.Errorf("error updating user followers count, %v", err)
	}

	return nil
}
