package userdom

import "github.com/google/uuid"

// SystemActorUserID is the fixed UUID of the built-in agent-bot user
// (seeded by bootstrap.seedAgentBotUser). It is used to attribute automated
// actions that have no human actor — e.g. the automation-workflow engine
// reassigning a task — for pipelines (like NotificationConsumer) that require
// a valid actor_user_id.
var SystemActorUserID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
