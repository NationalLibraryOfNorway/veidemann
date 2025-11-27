/*
 * Copyright 2017 National Library of Norway.
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
package no.nb.nna.veidemann.commons;

import no.nb.nna.veidemann.api.commons.v1.Error;
import no.nb.nna.veidemann.api.commons.v1.ErrorOrBuilder;

/**
 * Status codes in addition to those defined by http.
 * <p>
 * This list is based on error codes from Heritrix
 */
public enum ExtraStatusCodes {
    /**
     * SUCCESSFUL_DNS (1)
     * <p>
     * Successful DNS lookup
     */
    SUCCESSFUL_DNS(1, "Successful DNS lookup", false),
    /**
     * NEVER_TRIED (0)
     * <p>
     * Fetch never tried (perhaps protocol unsupported or illegal URI)
     */
    NEVER_TRIED(0, "Fetch never tried (perhaps protocol unsupported or illegal URI)", false),
    /**
     * FAILED_DNS (-1)
     * <p>
     * DNS lookup failed
     */
    FAILED_DNS(-1, "DNS lookup failed", true),
    /**
     * CONNECT_FAILED (-2)
     * <p>
     * HTTP connect failed
     */
    CONNECT_FAILED(-2, "HTTP connect failed", true),
    /**
     * CONNECT_BROKEN (-3)
     * <p>
     * HTTP connect broken
     */
    CONNECT_BROKEN(-3, "HTTP connect broken", true),
    /**
     * HTTP_TIMEOUT (-4)
     * <p>
     * HTTP timeout
     */
    HTTP_TIMEOUT(-4, "HTTP timeout", true),
    /**
     * RUNTIME_EXCEPTION (-5)
     * <p>
     * Unexpected runtime exception.  See runtime-errors.log.
     */
    RUNTIME_EXCEPTION(-5, "Unexpected runtime exception.  See runtime-errors.log.", true),
    /**
     * DOMAIN_LOOKUP_FAILED (-6)
     * <p>
     * Prerequisite domain-lookup failed, precluding fetch attempt.
     */
    DOMAIN_LOOKUP_FAILED(-6, "Prerequisite domain-lookup failed, precluding fetch attempt.", true),
    /**
     * ILLEGAL_URI (-7)
     * <p>
     * URI recognized as unsupported or illegal.
     */
    ILLEGAL_URI(-7, "URI recognized as unsupported or illegal.", false),
    /**
     * RETRY_LIMIT_REACHED (-8)
     * <p>
     * Multiple retries failed, retry limit reached.
     */
    RETRY_LIMIT_REACHED(-8, "Multiple retries failed, retry limit reached.", false),
    /**
     * FAILED_FETCHING_ROBOTS (-61)
     * <p>
     * Prerequisite robots.txt fetch failed, precluding a fetch attempt.
     */
    FAILED_FETCHING_ROBOTS(-61, "Prerequisite robots.txt fetch failed, precluding a fetch attempt.", true),
    /**
     * EMPTY_RESPONSE (-404)
     * <p>
     * Empty HTTP response interpreted as a 404.
     */
    EMPTY_RESPONSE(-404, "Empty HTTP response interpreted as a 404.", true),
    /**
     * SEVERE (-3000)
     * <p>
     * Severe Java Error condition occured such as OutOfMemoryError or StackOverflowError during URI processing.
     */
    SEVERE(-3000, "Severe Java Error condition occured such as OutOfMemoryError or StackOverflowError during URI processing.", true),
    /**
     * CHAFF_DETECTION (-4000)
     * <p>
     * Chaff detection of traps/content with negligible value applied.
     */
    CHAFF_DETECTION(-4000, "Chaff detection of traps/content with negligible value applied.", false),
    /**
     * TOO_MANY_HOPS (-4001)
     * <p>
     * The URI is too many link hops away from the seed.
     */
    TOO_MANY_HOPS(-4001, "The URI is too many link hops away from the seed.", false),
    /**
     * TOO_MANY_TRANSITIVE_HOPS (-4002)
     * <p>
     * The URI is too many embed/transitive hops away from the last URI in scope.
     */
    TOO_MANY_TRANSITIVE_HOPS(-4002, "The URI is too many embed/transitive hops away from the last URI in scope.", false),
    /**
     * ALREADY_SEEN (-4100)
     * <p>
     * The URI is already fetched.
     * Might happen if two URI's after normalization points to the same resource.
     */
    ALREADY_SEEN(-4100, "The URI is already fetched. Might happen if two URI's after normalization points to the same resource.", false),
    /**
     * PRECLUDED_BY_SCOPE_CHANGE (-5000)
     * <p>
     * The URI is out of scope upon reexamination.  This only happens if the scope changes during the crawl.
     */
    PRECLUDED_BY_SCOPE_CHANGE(-5000, "The URI is out of scope upon reexamination.  This only happens if the scope changes during the crawl.", false),
    /**
     * BLOCKED (-5001)
     * <p>
     * Blocked from fetch by user setting.
     */
    BLOCKED(-5001, "Blocked from fetch by user setting.", false),
    /**
     * BLOCKED_BY_CUSTOM_PROCESSOR (-5002)
     * <p>
     * Blocked by a custom processor.
     */
    BLOCKED_BY_CUSTOM_PROCESSOR(-5002, "Blocked by a custom processor.", false),
    /**
     * BLOCKED_MIXED_CONTENT (-5010)
     * <p>
     * Blocked because insecure content was loaded from secure context.
     */
    BLOCKED_MIXED_CONTENT(-5010, "Blocked because insecure content was loaded from secure context.", false),
    /**
     * CANCELED_BY_BROWSER (-5011)
     * <p>
     * The browser driving the fetch canceled the request.
     */
    CANCELED_BY_BROWSER(-5011, "The browser driving the fetch canceled the request.", true),
    /**
     * QUOTA_EXCEEDED (-5003)
     * <p>
     * Blocked due to exceeding an established quota.
     */
    QUOTA_EXCEEDED(-5003, "Blocked due to exceeding an established quota.", false),
    /**
     * RUNTIME_EXCEEDED (-5004)
     * <p>
     * Blocked due to exceeding an established runtime
     */
    RUNTIME_EXCEEDED(-5004, "Blocked due to exceeding an established runtime", false),
    /**
     * DELETED_FROM_FRONTIER (-6000)
     * <p>
     * Deleted from Frontier by user.
     */
    DELETED_FROM_FRONTIER(-6000, "Deleted from Frontier by user.", false),
    /**
     * PRECLUDED_BY_ROBOTS (-9998)
     * <p>
     * Robots.txt rules precluded fetch.
     */
    PRECLUDED_BY_ROBOTS(-9998, "Robots.txt rules precluded fetch.", false);

    final int code;

    final String description;

    final boolean temporary;

    private ExtraStatusCodes(int code, String description, boolean temporary) {
        this.code = code;
        this.description = description;
        this.temporary = temporary;
    }

    public int getCode() {
        return code;
    }

    public String getDescription() {
        return description;
    }

    /**
     * Get the nature of this status.
     * <p>
     * A temporary status is one that might be retried.
     * If this returns false, then this is the final status for the fetch.
     *
     * @return true if this error is eligible for retry
     */
    public boolean isTemporary() {
        return temporary;
    }

    /**
     * Get {@link ExtraStatusCodes} from a status code number.
     * @param code the status code
     * @return the matching {@link ExtraStatusCodes} or null if none matches
     */
    public static ExtraStatusCodes fromCode(int code) {
        for (ExtraStatusCodes c : ExtraStatusCodes.values()) {
            if (code == c.code) {
                return c;
            }
        }
        return null;
    }

    /**
     * Get an {@link ExtraStatusCodes} from an fetch error
     * @param error the error to convert
     * @return the resulting {@link ExtraStatusCodes}
     */
    public static ExtraStatusCodes fromFetchError(ErrorOrBuilder error) {
        for (ExtraStatusCodes v : values()) {
            if (v.getCode() == error.getCode()) {
                return v;
            }
        }
        throw new IllegalArgumentException("Unknown error code: " + error.getCode());
    }

    public Error toFetchError() {
        return Error.newBuilder().setCode(code).build();
    }

    public Error toFetchError(String message) {
        return Error.newBuilder().setCode(code).setMsg(message).build();
    }

}
