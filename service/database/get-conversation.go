package database

import (
	"database/sql"
	"fmt"
)

type ConversationDetail struct {
	Conversation ConversationSummary `json:"conversation"`
	Messages     []Message           `json:"messages"`
}

func (db *appdbimpl) GetConversation(authID, conversationID string) (ConversationDetail, error) {
	var result ConversationDetail

	// get the Users conversations to verify membership
	conversations, err := db.GetMyConversations(authID)
	if err != nil {
		return result, err
	}

	found := false

	for _, conversation := range conversations {
		if conversation.ID == conversationID {
			result.Conversation = conversation
			found = true
			break
		}
	}

	if !found {
		return result, ErrConversationNotFound
	}

	_, err = db.c.Exec(
		`UPDATE message_receipts
	     SET delivered_at = COALESCE(delivered_at, datetime('now')),
	         read_at = COALESCE(read_at, datetime('now'))
	     WHERE user_id = ?
	       AND message_id IN (
	           SELECT id FROM messages
	           WHERE conversation_id = ?
	       )`,
		authID, conversationID,
	)
	if err != nil {
		return result, fmt.Errorf("mark messages read: %w", err)
	}

	// get messages, newest first
	rows, err := db.c.Query(
		`SELECT
            m.id, m.type, m.text_content,
            m.photo_mime_type, m.photo_data,
            m.created_at, m.reply_to_message_id,
            m.forwarded_from_message_id,
            u.id, u.name, u.photo_mime_type, u.photo_data, 
		 NOT EXISTS (
		    SELECT 1 FROM message_receipts mr
		    WHERE mr.message_id = m.id
		      AND mr.delivered_at IS NULL
		  ),
		 NOT EXISTS (
		    SELECT 1 FROM message_receipts mr
		    WHERE mr.message_id = m.id
		      AND mr.read_at IS NULL
		  )
		 FROM messages m
         JOIN users u ON u.id = m.sender_id
         WHERE m.conversation_id = ?
         ORDER BY m.created_at DESC, m.id DESC`,
		conversationID,
	)
	if err != nil {
		return result, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	result.Messages = make([]Message, 0)

	for rows.Next() {
		var message Message
		var text, messageMime sql.NullString
		var replyTo, forwarded sql.NullString
		var senderMime sql.NullString
		var messagePhoto, senderPhoto []byte

		var deliveredStatus int
		var readStatus int

		err := rows.Scan(
			&message.ID,
			&message.Type,
			&text,
			&messageMime,
			&messagePhoto,
			&message.CreatedAt,
			&replyTo,
			&forwarded,
			&message.Sender.ID,
			&message.Sender.Name,
			&senderMime,
			&senderPhoto,
			&deliveredStatus,
			&readStatus,
		)
		if err != nil {
			return result, fmt.Errorf("read message: %w", err)
		}

		message.ConversationID = conversationID
		message.Reactions = make([]any, 0)

		if text.Valid {
			message.Text = text.String
		}

		if messageMime.Valid && len(messagePhoto) > 0 {
			message.Photo = &Photo{
				MimeType: messageMime.String,
				Data:     messagePhoto,
			}
		}

		if senderMime.Valid && len(senderPhoto) > 0 {
			message.Sender.Photo = &Photo{
				MimeType: senderMime.String,
				Data:     senderPhoto,
			}
		}

		if replyTo.Valid {
			message.ReplyToMessageID = replyTo.String
		}

		if forwarded.Valid {
			message.ForwardedFromMessageID = forwarded.String
		}

		// set message delivery and read statuses
		message.Delivered = deliveredStatus == 1
		message.Read = readStatus == 1

		result.Messages = append(result.Messages, message)
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("iterate messages: %w", err)
	}

	return result, nil
}
