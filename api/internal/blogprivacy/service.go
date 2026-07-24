package blogprivacy

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yueli-official/foundation/go/privacy"

	"platform/gokit/privacycatalog"
)

type Service struct {
	db          *sql.DB
	ownerKey    privacy.OwnerKey
	runtime     *privacy.PostgresRuntime
	newsletter  privacy.Processing
	measurement privacy.Processing
	host        privacy.OwnerHost
}

func NewPostgres(ctx context.Context, db *sql.DB, instanceKey string, ownerKeys ...privacy.OwnerKey) (*Service, error) {
	catalog, err := privacy.Compile(Definition(ownerKeys...))
	if err != nil {
		return nil, err
	}
	runtime, err := privacy.NewPostgresRuntime(ctx, catalog, privacy.PostgresOptions{
		DB: db, InstanceKey: instanceKey,
	})
	if err != nil {
		return nil, err
	}
	newsletter, err := runtime.Purpose(NewsletterPurpose)
	if err != nil {
		return nil, err
	}
	measurement, err := runtime.Purpose(MeasurementPurpose)
	if err != nil {
		return nil, err
	}
	host, err := privacy.NewPostgresOwnerHost(
		ctx, catalog, privacy.PostgresOptions{DB: db, InstanceKey: instanceKey},
		privacy.OwnerExecutorFunc((&ownerExecutor{db: db, ownerKey: ownerKey(catalog)}).execute),
	)
	if err != nil {
		return nil, err
	}
	return &Service{
		db: db, ownerKey: ownerKey(catalog), runtime: runtime, newsletter: newsletter,
		measurement: measurement, host: host,
	}, nil
}

func (service *Service) OwnerHost() privacy.OwnerHost { return service.host }

// ReconcileNewsletter imports already-confirmed subscribers into the
// versioned ledger. It is idempotent and provides the migration path from the
// former status-only implementation.
func (service *Service) ReconcileNewsletter(ctx context.Context) error {
	rows, err := service.db.QueryContext(ctx, `
SELECT id::text, email, COALESCE(confirmed_at, created_at)
FROM subscribers WHERE status='confirmed'`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, email string
		var occurredAt time.Time
		if err := rows.Scan(&id, &email, &occurredAt); err != nil {
			return err
		}
		_, err := service.runtime.Evidence().Consent(ctx, privacy.ConsentCommand{
			IdempotencyKey: privacy.IdempotencyKey("newsletter-import:" + id + ":v1"),
			Subject:        service.subscriber(email), Notice: newsletterNotice(),
			Purposes:   []privacy.PurposeRef{service.newsletter.Ref()},
			OccurredAt: occurredAt, Channel: "legacy_double_opt_in",
			EvidenceDigest: digest("subscriber:" + id),
		})
		if err != nil {
			return err
		}
	}
	return rows.Err()
}

// ConfirmSubscription atomically commits both the subscriber state and the
// versioned consent evidence using the caller-owned transaction seam.
func (service *Service) ConfirmSubscription(ctx context.Context, token string) (string, bool, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", false, nil
	}
	tx, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = tx.Rollback() }()

	var id, email, status string
	var occurredAt time.Time
	err = tx.QueryRowContext(ctx, `
SELECT id::text, email, status, COALESCE(confirmed_at, now())
FROM subscribers WHERE confirm_token=$1 FOR UPDATE`, token).
		Scan(&id, &email, &status, &occurredAt)
	if errors.Is(err, sql.ErrNoRows) || status == "unsubscribed" {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if status == "pending" {
		occurredAt = time.Now().UTC().Truncate(time.Microsecond)
		if _, err := tx.ExecContext(ctx, `
UPDATE subscribers SET status='confirmed', confirmed_at=$2 WHERE id=$1::uuid`,
			id, occurredAt); err != nil {
			return "", false, err
		}
	}
	bound := service.runtime.Bind(tx)
	if _, err := bound.Evidence().Consent(ctx, privacy.ConsentCommand{
		IdempotencyKey: privacy.IdempotencyKey("newsletter-confirm:" + id + ":v1"),
		Subject:        service.subscriber(email), Notice: newsletterNotice(),
		Purposes:   []privacy.PurposeRef{service.newsletter.Ref()},
		OccurredAt: occurredAt, Channel: "double_opt_in",
		EvidenceDigest: digest(token),
	}); err != nil {
		return "", false, err
	}
	if err := tx.Commit(); err != nil {
		return "", false, err
	}
	return email, true, nil
}

func (service *Service) Unsubscribe(ctx context.Context, token string) (bool, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return false, nil
	}
	tx, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	var id, email, status string
	var occurredAt time.Time
	err = tx.QueryRowContext(ctx, `
SELECT id::text, email, status, COALESCE(unsubscribed_at, now())
FROM subscribers WHERE confirm_token=$1 FOR UPDATE`, token).
		Scan(&id, &email, &status, &occurredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if status != "unsubscribed" {
		occurredAt = time.Now().UTC().Truncate(time.Microsecond)
		if _, err := tx.ExecContext(ctx, `
UPDATE subscribers SET status='unsubscribed', unsubscribed_at=$2 WHERE id=$1::uuid`,
			id, occurredAt); err != nil {
			return false, err
		}
	}
	bound := service.runtime.Bind(tx)
	if _, err := bound.Evidence().Withdraw(ctx, privacy.WithdrawalCommand{
		IdempotencyKey: privacy.IdempotencyKey("newsletter-withdraw:" + id + ":v1"),
		Subject:        service.subscriber(email), Purposes: []privacy.PurposeRef{service.newsletter.Ref()},
		OccurredAt: occurredAt, Channel: "unsubscribe_link", Reason: "subscriber_request",
	}); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (service *Service) CanDeliverNewsletter(ctx context.Context, email string) (bool, error) {
	decision, err := service.newsletter.Decide(ctx, privacy.DecisionInput{
		Subject: privacy.SingleSubject(service.subscriber(email)),
	})
	return err == nil && decision.Allows(), err
}

// CanMeasure maps only the explicit Sec-GPC: 1 signal. Its absence is not
// represented as consent or a negative signal.
func (service *Service) CanMeasure(ctx context.Context, gpc bool) (bool, error) {
	input := privacy.DecisionInput{}
	if gpc {
		input.Signals = []privacy.ObservedSignal{{Signal: GPCSignal, AssertedAt: time.Now().UTC()}}
	}
	decision, err := service.measurement.Decide(ctx, input)
	return err == nil && decision.Allows(), err
}

func newsletterNotice() privacy.NoticeRef {
	return privacy.NoticeRef{Key: NewsletterNotice, Revision: 1}
}

func (service *Service) subscriber(email string) privacy.SubjectRef {
	return privacy.SubjectRef{
		Owner: service.ownerKey, Kind: privacycatalog.SubscriberSubject,
		Value: strings.ToLower(strings.TrimSpace(email)),
	}
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

type ownerExecutor struct {
	db       *sql.DB
	ownerKey privacy.OwnerKey
}

func (executor *ownerExecutor) execute(ctx context.Context, instruction privacy.OwnerInstruction) (privacy.OwnerOutcome, error) {
	if instruction.Command.Operation != privacy.RightErasure {
		return privacy.OwnerOutcome{}, fmt.Errorf("blog privacy: unsupported operation %q", instruction.Command.Operation)
	}
	tx, err := executor.db.BeginTx(ctx, nil)
	if err != nil {
		return privacy.OwnerOutcome{}, err
	}
	defer func() { _ = tx.Rollback() }()
	results := make([]privacy.DatasetOutcome, 0, len(instruction.Command.Datasets))
	for _, dataset := range instruction.Command.Datasets {
		outcome, err := eraseBlogDataset(ctx, tx, executor.ownerKey, dataset, instruction.Command.Subject)
		if err != nil {
			return privacy.OwnerOutcome{}, err
		}
		results = append(results, outcome)
	}
	if err := tx.Commit(); err != nil {
		return privacy.OwnerOutcome{}, err
	}
	return privacy.OwnerOutcome{Terminal: true, Results: results}, nil
}

func eraseBlogDataset(
	ctx context.Context, tx *sql.Tx, ownerKey privacy.OwnerKey,
	dataset privacy.DatasetKey, subjects privacy.SubjectContext,
) (privacy.DatasetOutcome, error) {
	userIDs, emails := subjectValues(ownerKey, subjects)
	switch dataset {
	case privacycatalog.BlogNewsletterDataset:
		count, err := execForValues(ctx, tx, `DELETE FROM subscribers WHERE lower(email)=$1`, emails)
		return deletedOutcome(dataset, count), err
	case privacycatalog.BlogCommentsDataset:
		count, err := execForValues(ctx, tx, `
UPDATE comments SET user_id='', author_name='已注销用户', author_email='', ip='', user_agent=''
WHERE user_id=$1 AND deleted_at IS NULL`, userIDs)
		return anonymizedOutcome(dataset, count), err
	case privacycatalog.BlogAuthorshipDataset:
		var count int64
		for _, id := range userIDs {
			anonymous := "deleted:" + digest(id)[:24]
			for _, query := range []string{
				`UPDATE posts SET author_id=$2 WHERE author_id=$1`,
				`UPDATE post_revisions SET author_id=$2 WHERE author_id=$1`,
			} {
				result, err := tx.ExecContext(ctx, query, id, anonymous)
				if err != nil {
					return privacy.DatasetOutcome{}, err
				}
				n, _ := result.RowsAffected()
				count += n
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM author_profiles WHERE author_id=$1`, id); err != nil {
				return privacy.DatasetOutcome{}, err
			}
		}
		return anonymizedOutcome(dataset, count), nil
	case privacycatalog.BlogReactionsDataset:
		var count int64
		for _, query := range []string{
			`DELETE FROM post_likes WHERE user_id=$1`,
			`DELETE FROM post_bookmarks WHERE user_id=$1`,
		} {
			n, err := execForValues(ctx, tx, query, userIDs)
			if err != nil {
				return privacy.DatasetOutcome{}, err
			}
			count += n
		}
		return deletedOutcome(dataset, count), nil
	default:
		return privacy.DatasetOutcome{}, fmt.Errorf("blog privacy: unknown dataset %q", dataset)
	}
}

func subjectValues(ownerKey privacy.OwnerKey, subjects privacy.SubjectContext) (users, emails []string) {
	all := append([]privacy.SubjectRef(nil), subjects.Aliases...)
	if subjects.Current != nil {
		all = append(all, *subjects.Current)
	}
	for _, subject := range all {
		if subject.Owner != ownerKey {
			continue
		}
		switch subject.Kind {
		case privacycatalog.UserSubject:
			users = append(users, subject.Value)
		case privacycatalog.SubscriberSubject:
			emails = append(emails, strings.ToLower(strings.TrimSpace(subject.Value)))
		}
	}
	return users, emails
}

func ownerKey(catalog *privacy.Catalog) privacy.OwnerKey {
	owner, _ := catalog.Owner()
	return owner.Ref.Key
}

func execForValues(ctx context.Context, tx *sql.Tx, query string, values []string) (int64, error) {
	var count int64
	for _, value := range values {
		result, err := tx.ExecContext(ctx, query, value)
		if err != nil {
			return 0, err
		}
		n, _ := result.RowsAffected()
		count += n
	}
	return count, nil
}

func deletedOutcome(dataset privacy.DatasetKey, count int64) privacy.DatasetOutcome {
	disposition := privacy.DispositionDeleted
	if count == 0 {
		disposition = privacy.DispositionNotFound
	}
	return privacy.DatasetOutcome{Dataset: dataset, Disposition: disposition, Count: count}
}

func anonymizedOutcome(dataset privacy.DatasetKey, count int64) privacy.DatasetOutcome {
	disposition := privacy.DispositionAnonymized
	if count == 0 {
		disposition = privacy.DispositionNotFound
	}
	return privacy.DatasetOutcome{Dataset: dataset, Disposition: disposition, Count: count}
}
