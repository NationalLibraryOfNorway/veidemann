/*
 * Copyright 2026 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 */
package no.nb.nna.veidemann.frontier.db;

/** Result of deleting queued work for an execution. */
public record QueueCleanupResult(long deleted, long preservedActive) {
}
