-- Remove the restrictive check on event_source so the OpenHands SDK can emit
-- any source value (e.g. 'environment', 'task', etc.) without violating the constraint.
ALTER TABLE agent_conversation_events
    DROP CONSTRAINT IF EXISTS agent_conversation_events_event_source_check;
