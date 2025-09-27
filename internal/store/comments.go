package store

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/oseayemenre/pagesy/internal/models"
	"time"
)

var (
	ErrCommentNotFound = errors.New("comment not found")
)

// get the comments, get comments by recent, best, most liked, comment category and entity,by comment
func (s *PostgresStore) GetComment(params models.HandlePostCommentParameters) ([]models.HandleGetCommentVotes, error) {
	if params.Limit == 0 {
		params.Limit = 15
	}
	commentQuery := `
		SELECT 
				c.id, c.category, c.entity_type, c.entity_id, c.user_id, c.content, c.is_author, c.is_post, c.is_exclusive, c.pinned,
				c.vote_id, c.image, c.seen, c.is_deleted, c.created_at, c.modified_at, COALESCE(v.vote,0) as vote_value, COUNT(v.vote) 
                OVER (PARTITION BY c.id, v.vote) as vote_count
		FROM comments c
		LEFT JOIN votes v ON c.id = v.comment_id
		WHERE c.category = $1`
	args := []interface{}{params.Category}
	argIndex := 2
	if params.EntityType != nil {
		commentQuery += fmt.Sprintf(` AND c.entity_type = $%d`, argIndex)
		args = append(args, *params.EntityType)
		argIndex++
	}
	if params.EntityId != nil {
		commentQuery += fmt.Sprintf(` AND c.entity_id = $%d`, argIndex)
		args = append(args, *params.EntityId)
		argIndex++
	}
	sortedCommentQuery := `WITH comments AS (` + commentQuery + `) SELECT * FROM comments`
	switch params.SortBy {
	case "best":
		sortedCommentQuery += ` ORDER BY pinned DESC, (SELECT COUNT(*) FROM votes WHERE comment_id = id AND vote = 1) - (SELECT COUNT(*) FROM votes WHERE comment_id = id AND vote = 2) DESC, created_at DESC`
	case "most_liked":
		sortedCommentQuery += ` ORDER BY pinned DESC, (SELECT COUNT(*) FROM votes where comment_id = id AND vote = 1) DESC, created_at DESC`
	case "least_liked":
		sortedCommentQuery += ` ORDER BY pinned DESC, (SELECT COUNT(*) FROM votes WHERE comment_id = id AND vote = 2) DESC, created_at DESC`
	default:
		sortedCommentQuery += " ORDER BY pinned DESC, c.created_at DESC"
	}

	sortedCommentQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset)

	rows, err := s.DB.Query(sortedCommentQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query comments: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
		}
	}(rows)

	var comments []models.Comment
	CommentIDs := make([]uuid.UUID, 0)

	for rows.Next() {
		var comment models.Comment
		var voteValue, voteCount int

		err := rows.Scan(
			&comment.Id, &comment.Category, &comment.Entity_type, &comment.Entity_id, &comment.User_id, &comment.Content,
			&comment.Isauthor, &comment.IsPost, &comment.Isexcluive, &comment.Pinned,
			&comment.Vote_id, &comment.Image, &comment.Seen, &comment.Is_deleted, &comment.Created_at, &comment.Modified_at,
			&voteValue, &voteCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, comment)
		CommentIDs = append(CommentIDs, comment.Id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan comments: %w", err)
	}
	if len(CommentIDs) == 0 {
		return nil, ErrCommentNotFound
	}

	voteQuery := `
        SELECT comment_id, vote, COUNT(*) as count
        FROM votes 
        WHERE comment_id = ANY($1)
        GROUP BY comment_id, vote`
	voteRows, err := s.DB.Query(voteQuery, CommentIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query votes: %w", err)
	}
	defer func(voteRows *sql.Rows) {
		err := voteRows.Close()
		if err != nil {
		}
	}(voteRows)

	voteMap := make(map[uuid.UUID]map[int]int)
	for voteRows.Next() {
		var commentID uuid.UUID
		var vote, count int

		err := voteRows.Scan(&commentID, &vote, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vote: %w", err)
		}

		if voteMap[commentID] == nil {
			voteMap[commentID] = make(map[int]int)
		}
		voteMap[commentID][vote] = count
	}

	if err = voteRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over vote rows: %w", err)
	}

	result := make([]models.HandleGetCommentVotes, len(comments))
	for i, comment := range comments {
		voteCounts := voteMap[comment.Id]
		if voteCounts == nil {
			voteCounts = make(map[int]int)
		}

		totalVotes := 0
		for _, count := range voteCounts {
			totalVotes += count
		}

		result[i] = models.HandleGetCommentVotes{
			Comment:    comment,
			VoteCounts: voteCounts,
			TotalVotes: totalVotes,
		}
	}

	return result, nil
}

// validate if userid is in the system
// post comment
func (s *PostgresStore) PostComment(params models.HandlePostCommentParams) (*models.Comment, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func(tx *sql.Tx) {
		err := tx.Rollback()
		if err != nil {

		}
	}(tx)
	if params.Category != "Comments" && params.Category != "Forum" {
		return nil, fmt.Errorf("must be 'Comments' or 'Forum'")
	}

	var entityID *uuid.UUID
	var entityType *string

	if params.ParentID == nil {
		if params.EntityID == nil || params.EntityType == nil {
			return nil, fmt.Errorf("entity_id and entity_type are required for fresh comments")
		}

		if params.Category == "Comments" && *params.EntityType != "Chapters" {
			return nil, fmt.Errorf("for Comments category, entity_type must be 'Chapters'")
		}
		if params.Category == "Forum" && *params.EntityType != "Books" {
			return nil, fmt.Errorf("for Forum category, entity_type must be 'Books'")
		}

		if *params.EntityType == "Chapters" {
			var exists bool
			err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM chapters WHERE id = $1)", *params.EntityID).Scan(&exists)
			if err != nil {
				return nil, fmt.Errorf("failed to check chapter existence: %w", err)
			}
			if !exists {
				return nil, fmt.Errorf("chapter with ID %s does not exist", params.EntityID.String())
			}
		} else if *params.EntityType == "Books" {
			var exists bool
			err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM books WHERE id = $1)", *params.EntityID).Scan(&exists)
			if err != nil {
				return nil, fmt.Errorf("failed to check book existence: %w", err)
			}
			if !exists {
				return nil, fmt.Errorf("book with ID %s does not exist", params.EntityID.String())
			}
		}

		entityID = params.EntityID
		entityType = params.EntityType
	} else {
		var parentExists bool
		err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM comments WHERE id = $1 AND is_deleted = false)", *params.ParentID).Scan(&parentExists)
		if err != nil {
			return nil, fmt.Errorf("failed to check parent comment existence: %w", err)
		}
		if !parentExists {
			return nil, fmt.Errorf("parent comment with ID %s does not exist or is deleted", params.ParentID.String())
		}

		entityID = params.ParentID
		commentEntityType := "Comments"
		entityType = &commentEntityType
	}

	if params.IsAuthor {
		var bookID uuid.UUID
		if *entityType == "Books" {
			bookID = *entityID
		} else if *entityType == "Chapters" {
			err = tx.QueryRow("SELECT book_id FROM chapters WHERE id = $1", *entityID).Scan(&bookID)
			if err != nil {
				return nil, fmt.Errorf("failed to get book_id for chapter: %w", err)
			}
		} else {
			return nil, fmt.Errorf("cannot verify authorship for entity type %s", *entityType)
		}

		var authorID uuid.UUID
		err = tx.QueryRow("SELECT author_id FROM books WHERE id = $1", bookID).Scan(&authorID)
		if err != nil {
			return nil, fmt.Errorf("failed to get book author: %w", err)
		}

		if authorID != params.UserID {
			return nil, fmt.Errorf("user is not the author of this book")
		}
	}

	commentID := uuid.New()
	now := time.Now()

	query := `
		INSERT INTO comments (
			id, category, user_id, content, is_author, is_exclusive, 
			is_post, pinned, entity_id, entity_type, image, created_at, modified_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	_, err = tx.Exec(query,
		commentID, params.Category, params.UserID, params.Content,
		params.IsAuthor, params.IsExclusive, params.IsPost, params.Pinned,
		entityID, entityType, params.Image, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert comment: %w", err)
	}

	var comment models.Comment
	retrieveQuery := `
		SELECT id, category, user_id, content, is_author, is_exclusive, 
			   is_post, pinned, entity_id, entity_type, vote_id, image, 
			   seen, is_deleted, created_at, modified_at
		FROM comments 
		WHERE id = $1`

	err = tx.QueryRow(retrieveQuery, commentID).Scan(
		&comment.Id, &comment.Category, &comment.User_id, &comment.Content,
		&comment.Isauthor, &comment.Isexcluive, &comment.IsPost, &comment.Pinned,
		&comment.Entity_id, &comment.Entity_type, &comment.Vote_id, &comment.Image,
		&comment.Seen, &comment.Is_deleted, &comment.Created_at, &comment.Modified_at,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created comment: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &comment, nil
}

// edit comment
func (s *PostgresStore) EditComment(params models.HandleEditCommentParams) (*models.Comment, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func(tx *sql.Tx) {
		err := tx.Rollback()
		if err != nil {

		}
	}(tx)
	var commentUserID uuid.UUID
	var isDeleted bool
	err = tx.QueryRow(
		"SELECT user_id, is_deleted FROM comments WHERE id = $1",
		params.CommentID,
	).Scan(&commentUserID, &isDeleted)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("comment with ID %s not found", params.CommentID.String())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check comment ownership: %w", err)
	}

	if isDeleted {
		return nil, fmt.Errorf("cannot edit deleted comment")
	}

	if commentUserID != params.UserID {
		return nil, fmt.Errorf("user does not have permission to edit this comment")
	}

	_, err = tx.Exec(
		"UPDATE comments SET content = $1, image = $2, modified_at = CURRENT_TIMESTAMP WHERE id = $3",
		params.Content, params.Image, params.CommentID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}

	var comment models.Comment
	retrieveQuery := `
		SELECT id, category, user_id, content, is_author, is_exclusive, 
			   is_post, pinned, entity_id, entity_type, vote_id, image, 
			   seen, is_deleted, created_at, modified_at
		FROM comments 
		WHERE id = $1`

	err = tx.QueryRow(retrieveQuery, params.CommentID).Scan(
		&comment.Id, &comment.Category, &comment.User_id, &comment.Content,
		&comment.Isauthor, &comment.Isexcluive, &comment.IsPost, &comment.Pinned,
		&comment.Entity_id, &comment.Entity_type, &comment.Vote_id, &comment.Image,
		&comment.Seen, &comment.Is_deleted, &comment.Created_at, &comment.Modified_at,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated comment: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &comment, nil
}

// pin comments
func (s *PostgresStore) PinComment(params models.HandlePinCommentParams) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var entityType string
	var entityID uuid.UUID
	var isDeleted bool
	err = tx.QueryRow(
		"SELECT entity_type, entity_id, is_deleted FROM comments WHERE id = $1",
		params.CommentID,
	).Scan(&entityType, &entityID, &isDeleted)

	if err == sql.ErrNoRows {
		return fmt.Errorf("comment with ID %s not found", params.CommentID.String())
	}
	if err != nil {
		return fmt.Errorf("failed to get comment details: %w", err)
	}

	if isDeleted {
		return fmt.Errorf("cannot pin deleted comment")
	}
	if entityType == "Comments" {
		return fmt.Errorf("cannot pin comment replies")
	}

	var bookID uuid.UUID
	switch entityType {
	case "Books":
		bookID = entityID
	case "Chapters":
		err = tx.QueryRow("SELECT book_id FROM chapters WHERE id = $1", entityID).Scan(&bookID)
		if err != nil {
			return fmt.Errorf("failed to get book_id for chapter: %w", err)
		}
	default:
		return fmt.Errorf("cannot pin comment with entity type %s", entityType)
	}

	var authorID uuid.UUID
	err = tx.QueryRow("SELECT author_id FROM books WHERE id = $1", bookID).Scan(&authorID)
	if err != nil {
		return fmt.Errorf("failed to get book author: %w", err)
	}

	if authorID != params.AuthorID {
		return fmt.Errorf("user is not the author of this book and cannot pin comments")
	}

	_, err = tx.Exec(
		"UPDATE comments SET pinned = true, modified_at = CURRENT_TIMESTAMP WHERE id = $1",
		params.CommentID,
	)
	if err != nil {
		return fmt.Errorf("failed to pin comment: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// delete comment
func (s *PostgresStore) DeleteComment(params models.HandleDeleteCommentParams) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var commentUserID uuid.UUID
	var isDeleted bool
	err = tx.QueryRow(
		"SELECT user_id, is_deleted FROM comments WHERE id = $1",
		params.CommentID,
	).Scan(&commentUserID, &isDeleted)

	if err == sql.ErrNoRows {
		return fmt.Errorf("comment with ID %s not found", params.CommentID.String())
	}
	if err != nil {
		return fmt.Errorf("failed to check comment ownership: %w", err)
	}

	if isDeleted {
		return fmt.Errorf("comment is already deleted")
	}

	if commentUserID != params.UserID {
		return fmt.Errorf("user does not have permission to delete this comment")
	}

	_, err = tx.Exec(
		"UPDATE comments SET is_deleted = true, modified_at = CURRENT_TIMESTAMP WHERE id = $1",
		params.CommentID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
