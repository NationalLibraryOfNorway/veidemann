/*
 * Copyright 2026 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package no.nb.nna.veidemann.frontier.worker;

import no.nb.nna.veidemann.commons.db.DbException;

/** Raised when asynchronous work tries to queue or issue a terminal execution. */
public class CrawlExecutionNotActiveException extends DbException {
    private static final long serialVersionUID = 1L;

    public CrawlExecutionNotActiveException(String executionId) {
        super("Crawl execution '" + executionId + "' is no longer active");
    }
}
