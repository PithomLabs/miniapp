# Secret Rotation Without Exposure

## Fly secrets (development and production)

1. Provision the replacement credential at the provider (Neon password
   rotation, new token). Keep the old credential valid during the
   changeover where the provider allows overlap.
2. Stage the new value without deploying:
   `secrets:stage SECRET=DATABASE_URL` (hidden input, stdin import).
3. Confirm the staged state with `secrets:list` (metadata only).
4. Apply: `secrets:apply`. Machines restart with the new value;
   confirm the restart implication shown before applying.
5. Verify health (`fly:status` / `release:verify`) and application
   behavior before retiring anything.
6. Retire the old credential at the provider.

Production additionally requires the configured application,
matching organization, and typed application-name confirmation at
both staging and apply steps.

## Local development secrets

Local credentials rotate by replacing the file contents with owner-only
permissions preserved:

```sh
# Generate a new value without echoing it into history or logs.
openssl rand -base64 33 | tr -d '\n' > "$MINIAPP_SECRETS_DIR/db_password.new"
chmod 600 "$MINIAPP_SECRETS_DIR/db_password.new"
mv "$MINIAPP_SECRETS_DIR/db_password.new" "$MINIAPP_SECRETS_DIR/db_password"
```

Then restart local services and update the database role password to
match. Never reuse a rotated-out value, and never copy a Fly secret
into local development (or the reverse).

## What rotation never does

- Print, log, or persist secret values.
- Trigger an undeployed release implicitly (staging and applying are
  separate steps; rotation completes only at an explicit apply).
- Touch production without the configured target and confirmations.
