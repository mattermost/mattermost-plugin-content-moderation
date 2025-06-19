# Pre-Notification Content Moderation Enhancement Design Proposal

## Problem Statement

The current content moderation plugin has a critical flaw: it moderates posts **after** they are created using the `MessageHasBeenPosted` hook, but notifications containing harmful content are sent immediately when the post is created. Since notifications cannot be "recalled" once sent, users receive notifications with potentially harmful content before the moderation system can flag and remove the offending posts.

## Proposed Solution

Implement a multi-step moderation system that intercepts content at two key hook points and leverages background processing to minimize harmful notification delivery while maintaining system performance through graceful degradation.

### Core Strategy

The solution implements a three-step strategy with two interception points that progressively catches harmful content at different stages of the post lifecycle. The two hook interceptions have very short timeouts to maintain system responsiveness, while the third step relies on the existing background processor to handle cleanup without time constraints.

#### Complete Post Flow (Three Steps)

**Step 1: Pre-Creation Moderation (`MessageWillBePosted` Hook)**
1. User submits a post for creation
2. Hook intercepts the post before it's created
3. Post is added to status tracking as `PENDING`
4. Asynchronous moderation is initiated immediately
5. System waits up to **50ms** for moderation result
6. **If result received within 50ms:**
   - **Approved**: Post is created normally
   - **Flagged**: Post creation is blocked, user gets rejection message
7. **If timeout reached (50ms):**
   - Post is allowed to be created (fail-open for performance)
   - Status remains `PENDING` for next steps
   - Async moderation continues in background

**Step 2: Pre-Notification Filtering (`NotificationWillBePushed` Hook)**
1. Post has been created and system is about to send notifications
2. Hook intercepts each notification before delivery
3. System checks post status from Step 1
4. **If status is `APPROVED`:**
   - Notification is sent immediately
5. **If status is `FLAGGED`:**
   - Notification is blocked immediately (no timeout needed)
6. **If status is still `PENDING`:**
   - System waits up to **50ms** for moderation result
   - **If result received within 50ms:**
     - **Approved**: Notification is sent
     - **Flagged**: Notification is blocked
   - **If timeout reached (50ms):**
     - Notification is allowed to be sent (fail-open for performance)
     - Status remains `PENDING` for next step
     - Async moderation continues in background

**Step 3: Background Processor Cleanup**
1. Posts from Steps 1 and 2 that were created with `PENDING` status continue to be moderated in the background
2. The existing background processor thread waits for moderation results without timeout constraints
3. **When moderation completes:**
   - **If approved**: Status is updated to `APPROVED`, no further action needed
   - **If flagged**: Post is deleted (if it exists) and moderation event is reported to users
4. **This approach provides:**
   - No additional hook complexity or timing constraints
   - Leverages existing background processing infrastructure
   - Ensures all posts eventually get processed regardless of service speed
   - Maintains the existing fail-closed behavior for the background system

#### Cascading Protection Strategy

The three-step approach creates multiple opportunities to catch harmful content, with each step serving as a safety net for the previous one:

1. **Best case (< 50ms moderation)**: Harmful content never gets created or notified
2. **Good case (< 100ms moderation)**: Content gets created but notifications are blocked and post is removed when moderation completes
3. **Worst case (> 100ms moderation)**: Some notifications may go out, but post is eventually removed when moderation completes
