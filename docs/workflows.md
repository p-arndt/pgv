# Workflows

PGV is built to handle the messiest parts of local database development. Here are common scenarios where PGV shines.

## Scenario 1: The "Destructive Migration" Test

You are about to run a massive data migration that deletes columns, transforms data, and takes 5 minutes to run. You want to test it locally, but if it fails, you don't want to spend 20 minutes running `pg_restore` to fix your local DB.

1. **Save your state:**
   ```bash
   pgv checkpoint "before massive migration"
   ```
2. **Run your migration** (using Prisma, Drizzle, Flyway, etc).
3. **Did it fail? Roll back instantly:**
   ```bash
   pgv stop main
   pgv restore "before massive migration"
   pgv start main
   ```
   *Your database is back online exactly as it was, in 2 seconds.*

## Scenario 2: Parallel Feature Development

You are reviewing a PR for "Feature A" which introduces a new `orders` table. You are actively coding "Feature B" which introduces a `subscriptions` table. You need to switch between them without the schemas clashing.

1. **Create branches from your clean main snapshot:**
   ```bash
   pgv branch snap_clean feature-a
   pgv branch snap_clean feature-b
   ```
2. **Start both branches:**
   ```bash
   pgv start feature-a  # Runs on port 5541
   pgv start feature-b  # Runs on port 5542
   ```
3. **Switch contexts effortlessly:**
   You can point your application to `5541` to test the PR, and point it to `5542` to continue your own work. The databases are physically isolated.

## Scenario 3: Seed Data "Save States"

Generating massive amounts of seed data locally takes a long time. 

1. Write a script to generate 10GB of fake data. Run it on your `main` branch.
2. Save this state:
   ```bash
   pgv checkpoint "10GB seed data"
   ```
3. Now, whenever you need a fresh database full of data to test a new query, you can instantly branch off that snapshot without having to run the 10-minute seed script ever again:
   ```bash
   pgv branch "10GB seed data" query-test-env
   ```