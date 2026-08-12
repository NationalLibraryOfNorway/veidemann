package no.nb.nna.veidemann.frontier.testutil;

import com.google.protobuf.Timestamp;
import com.google.protobuf.util.Timestamps;
import no.nb.nna.veidemann.api.frontier.v1.CrawlHostGroup;
import org.assertj.core.api.AbstractAssert;
import org.assertj.core.api.Assertions;
import org.assertj.core.data.TemporalOffset;
import redis.clients.jedis.resps.Tuple;

import java.time.Instant;
import java.time.temporal.Temporal;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Map;

public class RedisDataAssert extends AbstractAssert<RedisDataAssert, RedisData> {
    public RedisDataAssert(RedisData actual) {
        super(actual, RedisDataAssert.class);
    }

    public static RedisDataAssert assertThat(RedisData actual) {
        return new RedisDataAssert(actual);
    }

    public RedisDataAssert hasQueueTotalCount(long expected) {
        long val = actual.getQueueCountTotal();
        if (val != expected) {
            failWithMessage("Expected total queue count to be <%d>, but was <%d>", expected, val);
        }
        return this;
    }

    public CrawlExecutionCountsAssert crawlExecutionQueueCounts() {
        return new CrawlExecutionCountsAssert(actual.getCrawlExecutionCounts());
    }

    public JobExecutionCountsAssert jobExecutionQueueCounts() {
        return new JobExecutionCountsAssert(actual.getJobExecutionCounts());
    }

    public CrawlExecutionJobExecutionsAssert crawlExecutionJobExecutions() {
        return new CrawlExecutionJobExecutionsAssert(actual.getCrawlExecutionJobExecutions());
    }

    public CrawlHostGroupMapAssert crawlHostGroups() {
        return new CrawlHostGroupMapAssert(actual.getCrawlHostGroups());
    }

    public SessionTokensAssert sessionTokens() {
        return new SessionTokensAssert(actual.getSessionTokens());
    }

    public DelayQueueAssert waitQueue() {
        return new DelayQueueAssert("wait queue", new ArrayList<>(actual.getWaitQueue()));
    }

    public DelayQueueAssert busyQueue() {
        return new DelayQueueAssert("busy queue", new ArrayList<>(actual.getBusyQueue()));
    }

    public ReadyQueueAssert readyQueue() {
        return new ReadyQueueAssert("ready queue", actual.getReadyQueue());
    }

    public static class CrawlHostGroupMapAssert extends AbstractAssert<CrawlHostGroupMapAssert, Map<String, CrawlHostGroup>> {
        public CrawlHostGroupMapAssert(Map<String, CrawlHostGroup> actual) {
            super(actual, CrawlHostGroupMapAssert.class);
        }

        public CrawlHostGroupMapAssert hasNumberOfElements(int expected) {
            if (actual.size() != expected) {
                failWithMessage("Expected number of CrawlHostGroups to be <%d>, but was <%d>",
                        expected, actual.size());
            }
            return this;
        }

        public CrawlHostGroupFromMapAssert id(String id) {
            return new CrawlHostGroupFromMapAssert(this, actual.get(id));
        }
    }

    public static class CrawlHostGroupFromMapAssert extends CrawlHostGroupAssert<CrawlHostGroupFromMapAssert> {
        final CrawlHostGroupMapAssert origin;

        public CrawlHostGroupFromMapAssert(CrawlHostGroupMapAssert origin, CrawlHostGroup actual) {
            super(actual);
            this.origin = origin;
        }

        public CrawlHostGroupFromMapAssert id(String id) {
            return origin.id(id);
        }
    }

    public static class CrawlExecutionCountsAssert extends AbstractAssert<CrawlExecutionCountsAssert, Map<String, Long>> {

        public CrawlExecutionCountsAssert(Map<String, Long> actual) {
            super(actual, CrawlExecutionCountsAssert.class);
        }

        public CrawlExecutionCountsAssert hasNumberOfElements(int expected) {
            if (actual.size() != expected) {
                failWithMessage("Expected number of CrawlExecution elements to be <%d>, but was <%d>",
                        expected, actual.size());
            }
            return this;
        }

        public CrawlExecutionCountsAssert hasQueueCount(String crawlExecutionId, long count) {
            if (actual.get(crawlExecutionId) != count) {
                failWithMessage("Expected count for CrawlExecution <%s> to be <%d>, but was <%d>", crawlExecutionId, count, actual.get(crawlExecutionId));
            }
            return this;
        }
    }

    public static class JobExecutionCountsAssert extends AbstractAssert<JobExecutionCountsAssert, Map<String, Long>> {

        public JobExecutionCountsAssert(Map<String, Long> actual) {
            super(actual, JobExecutionCountsAssert.class);
        }

        public JobExecutionCountsAssert hasNumberOfElements(int expected) {
            if (actual.size() != expected) {
                failWithMessage("Expected number of JobExecution elements to be <%d>, but was <%d>",
                        expected, actual.size());
            }
            return this;
        }

        public JobExecutionCountsAssert hasQueueCount(String jobExecutionId, long count) {
            Long actualCount = actual.get(jobExecutionId);
            if (actualCount == null || actualCount != count) {
                failWithMessage("Expected count for JobExecution <%s> to be <%d>, but was <%s>",
                        jobExecutionId, count, actualCount);
            }
            return this;
        }
    }

    public static class CrawlExecutionJobExecutionsAssert
            extends AbstractAssert<CrawlExecutionJobExecutionsAssert, Map<String, String>> {

        public CrawlExecutionJobExecutionsAssert(Map<String, String> actual) {
            super(actual, CrawlExecutionJobExecutionsAssert.class);
        }

        public CrawlExecutionJobExecutionsAssert hasNumberOfElements(int expected) {
            if (actual.size() != expected) {
                failWithMessage("Expected number of CrawlExecution ownership elements to be <%d>, but was <%d>",
                        expected, actual.size());
            }
            return this;
        }

        public CrawlExecutionJobExecutionsAssert mapsTo(String crawlExecutionId, String jobExecutionId) {
            String actualJobExecutionId = actual.get(crawlExecutionId);
            if (!jobExecutionId.equals(actualJobExecutionId)) {
                failWithMessage("Expected CrawlExecution <%s> to map to JobExecution <%s>, but was <%s>",
                        crawlExecutionId, jobExecutionId, actualJobExecutionId);
            }
            return this;
        }
    }

    public static class SessionTokensAssert extends AbstractAssert<SessionTokensAssert, Map<String, String>> {

        public SessionTokensAssert(Map<String, String> actual) {
            super(actual, SessionTokensAssert.class);
        }

        public SessionTokensAssert hasNumberOfElements(int expected) {
            if (actual.size() != expected) {
                failWithMessage("Expected number of session tokens to be <%d>, but was <%d>",
                        expected, actual.size());
            }
            return this;
        }

        public SessionTokensAssert hasCrawlHostId(String sessionToken, String crawlHostGroupId) {
            if (!actual.get(sessionToken).equals(crawlHostGroupId)) {
                failWithMessage("Expected CrawlHostGroupId for session token <%s> to be <%s>, but was <%s>", sessionToken, crawlHostGroupId, actual.get(sessionToken));
            }
            return this;
        }
    }

    public static class ReadyQueueAssert extends AbstractAssert<ReadyQueueAssert, List<String>> {
        final String name;

        public ReadyQueueAssert(String name, List<String> actual) {
            super(actual, ReadyQueueAssert.class);
            this.name = name;
        }

        public ReadyQueueAssert hasNumberOfElements(int expected) {
            if (actual.size() != expected) {
                failWithMessage("Expected number of elements for %s to be <%d>, but was <%d>",
                        name, expected, actual.size());
            }
            return this;
        }

        public ReadyQueueAssert containsExactly(String... elements) {
            Assertions.assertThat(actual)
                    .withFailMessage("Expected %s to contain exactly in order <%s>, but was <%s>", name, Arrays.toString(elements), actual)
                    .containsExactly(elements);
            return this;
        }
    }

    public static class DelayQueueAssert extends AbstractAssert<DelayQueueAssert, List<Tuple>> {
        final String name;

        public DelayQueueAssert(String name, List<Tuple> actual) {
            super(actual, DelayQueueAssert.class);
            this.name = name;
        }

        public DelayQueueAssert hasNumberOfElements(int expected) {
            if (actual.size() != expected) {
                failWithMessage("Expected number of elements for %s to be <%d>, but was <%d>",
                        name, expected, actual.size());
            }
            return this;
        }

        public DelayQueueElementAssert element(int elementNumber) {
            return new DelayQueueElementAssert(this, elementNumber, actual.get(elementNumber));
        }
    }

    public static class DelayQueueElementAssert extends AbstractAssert<DelayQueueElementAssert, Tuple> {
        final DelayQueueAssert origin;
        final int elementNumber;

        public DelayQueueElementAssert(DelayQueueAssert origin, int elementNumber, Tuple actual) {
            super(actual, DelayQueueElementAssert.class);
            this.origin = origin;
            this.elementNumber = elementNumber;
        }

        public DelayQueueElementAssert element(int elementNumber) {
            return origin.element(elementNumber);
        }

//        public DelayQueueElementAssert hasScore(double expected) {
//            if (actual.getScore() != expected) {
//                failWithMessage("Expected score for element #%d of %s to be <%f>, but was <%f>",
//                        elementNumber, origin.name, expected, actual.getScore());
//            }
//            return this;
//        }

        public DelayQueueElementAssert hasTimestamp(Timestamp expected) {
            double expectedScore = Long.valueOf(Timestamps.toMillis(expected)).doubleValue();
            if (actual.getScore() != expectedScore) {
                Instant expectedTime = Instant.ofEpochMilli(Double.valueOf(expectedScore).longValue());
                Instant actualTime = Instant.ofEpochMilli(Double.valueOf(actual.getScore()).longValue());
                failWithMessage("Expected timestamp for element #%d of %s to be <%s>, but was <%s>",
                        elementNumber, origin.name, expectedTime, actualTime);
            }
            return this;
        }

        public DelayQueueElementAssert hasTimestampCloseTo(Timestamp expected, TemporalOffset<? extends Temporal> offset) {
            return hasTimestampCloseToUnchecked(expected, offset);
        }

        @SuppressWarnings({"rawtypes", "unchecked"})
        private DelayQueueElementAssert hasTimestampCloseToUnchecked(Timestamp expected, TemporalOffset offset) {
            double expectedScore = Long.valueOf(Timestamps.toMillis(expected)).doubleValue();
            Instant expectedTime = Instant.ofEpochMilli(Double.valueOf(expectedScore).longValue());
            Instant actualTime = Instant.ofEpochMilli(Double.valueOf(actual.getScore()).longValue());
            if (offset.isBeyondOffset(actualTime, expectedTime)) {
                failWithMessage("Expected timestamp for element #%d of %s to be close to <%s> %s. Actual timestamp was <%s>",
                        elementNumber, origin.name, expectedTime, offset.getBeyondOffsetDifferenceDescription(expectedTime, actualTime), actualTime);
            }
            return this;
        }

        public DelayQueueElementAssert hasValue(String expected) {
            if (!actual.getElement().equals(expected)) {
                failWithMessage("Expected value for element #%d of %s to be <%s>, but was <%s>",
                        elementNumber, origin.name, expected, actual.getElement());
            }
            return this;
        }
    }
}
