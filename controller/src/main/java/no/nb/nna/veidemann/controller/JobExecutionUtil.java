package no.nb.nna.veidemann.controller;

import com.google.common.util.concurrent.FutureCallback;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.SettableFuture;
import io.grpc.Status;
import io.grpc.stub.StreamObserver;
import no.nb.nna.veidemann.api.commons.v1.Error;
import no.nb.nna.veidemann.api.commons.v1.FieldMask;
import no.nb.nna.veidemann.api.config.v1.Annotation;
import no.nb.nna.veidemann.api.config.v1.ConfigObject;
import no.nb.nna.veidemann.api.config.v1.Kind;
import no.nb.nna.veidemann.api.config.v1.ListRequest;
import no.nb.nna.veidemann.api.controller.v1.CrawlerStatus;
import no.nb.nna.veidemann.api.frontier.v1.CountResponse;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionId;
import no.nb.nna.veidemann.api.frontier.v1.CrawlHostGroup;
import no.nb.nna.veidemann.api.frontier.v1.ExecutionIds;
import no.nb.nna.veidemann.api.frontier.v1.JobExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.JobExecutionStatus.State;
import no.nb.nna.veidemann.api.frontier.v1.QueueCountsResponse;
import no.nb.nna.veidemann.api.report.v1.CrawlExecutionsListRequest;
import no.nb.nna.veidemann.api.report.v1.JobExecutionsListRequest;
import no.nb.nna.veidemann.commons.db.ChangeFeed;
import no.nb.nna.veidemann.commons.db.ConfigAdapter;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.DbService;
import no.nb.nna.veidemann.commons.util.ApiTools;
import no.nb.nna.veidemann.controller.ControllerApiServer.JobExecutionListener;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.OffsetDateTime;
import java.time.temporal.ChronoUnit;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.concurrent.CompletionService;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.ExecutorCompletionService;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.Semaphore;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.ThreadPoolExecutor.CallerRunsPolicy;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.atomic.AtomicReference;

import static no.nb.nna.veidemann.commons.util.ApiTools.buildLabel;

public class JobExecutionUtil {
    private static final Logger LOG = LoggerFactory.getLogger(JobExecutionUtil.class);

    public final static String SEED_TYPE_LABEL_KEY = "v7n_seed-type";
    private static final int MAX_OUTSTANDING_SEED_SUBMISSIONS = 5000;
    private static final long COMPLETION_POLL_MILLIS = 100L;
    private final static Map<String, FrontierClient> frontierClients = new HashMap<>();

    private final static ExecutorService exe = Executors.newFixedThreadPool(16);
    private final static ExecutorService submitSeedExecutor =
            new ThreadPoolExecutor(4, 16, 10L, TimeUnit.SECONDS,
                    new LinkedBlockingQueue<>(MAX_OUTSTANDING_SEED_SUBMISSIONS), new CallerRunsPolicy());

    private JobExecutionUtil() {
    }

    public static void addFrontierClient(String seedType, FrontierClient client) {
        frontierClients.put(seedType.toLowerCase(), client);
    }

    /**
     * Helper method for getting one object. Sends NOT_FOUND if responseSupplier returns null.
     *
     * @param responseSupplier the supplier which result is checked for null
     * @param responseObserver the observer to send the object to
     */
    public static <T> void handleGet(CheckedSupplier<T, DbException> responseSupplier, StreamObserver<T> responseObserver) {
        try {
            T response = responseSupplier.get();
            if (response == null) {
                Status status = Status.NOT_FOUND;
                responseObserver.onError(status.asException());
            } else {
                responseObserver.onNext(response);
                responseObserver.onCompleted();
            }
        } catch (Exception ex) {
            LOG.error(ex.getMessage(), ex);
            Status status = Status.UNKNOWN.withDescription(ex.toString());
            responseObserver.onError(status.asException());
        }
    }

    public static boolean crawlSeed(CompletionService<CrawlExecutionId> submitSeedCompletionService,
                                    ConfigObject job, ConfigObject seed,
                                    JobExecutionStatus jobExecutionStatus,
                                    OffsetDateTime timeout, boolean addToRunningJob) {
        if (seed.getSeed().getDisabled()) {
            LOG.debug("Seed '{}' is disabled", seed.getMeta().getName());
            return false;
        }

        if (addToRunningJob && isSeedInJobExecution(seed, jobExecutionStatus)) {
            LOG.debug("Seed '{}' already has a crawl execution in job execution '{}'",
                    seed.getMeta().getName(), jobExecutionStatus.getId());
            return false;
        }

        FrontierClient frontierClient = getFrontierClient(seed);
        if (frontierClient == null) {
            LOG.warn("No Frontier client is configured for seed '{}'", seed.getMeta().getName());
            return false;
        }

        LOG.debug("Submitting seed '{}' for job execution '{}'", seed.getMeta().getName(), jobExecutionStatus.getId());
        if (submitSeedCompletionService != null) {
            submitSeedCompletionService.submit(() -> frontierClient.crawlSeed(job, seed, jobExecutionStatus, timeout));
            return true;
        }

        try {
            CrawlExecutionId crawlExecutionId = frontierClient.crawlSeed(job, seed, jobExecutionStatus, timeout);
            return isAccepted(crawlExecutionId, seed);
        } catch (Exception e) {
            LOG.warn("Frontier rejected seed '{}' for job execution '{}'",
                    seed.getMeta().getName(), jobExecutionStatus.getId(), e);
            return false;
        }
    }

    public static JobExecutionStatus submitSeeds(ConfigObject job, JobExecutionStatus jobExecutionStatus,
                                                 OffsetDateTime timeout, boolean addToRunningJob,
                                                 List<JobExecutionListener> jobExecutionListeners) {
        SettableFuture<JobExecutionStatus> jobExecutionStatusFuture = SettableFuture.create();

        ListRequest.Builder seedRequest = ListRequest.newBuilder().setKind(Kind.seed);
        seedRequest.getQueryMaskBuilder()
                .addPaths(Kind.seed.name() + ".jobRef")
                .addPaths(Kind.seed.name() + ".disabled");
        seedRequest.getQueryTemplateBuilder().getSeedBuilder()
                .addJobRefBuilder().setKind(Kind.crawlJob).setId(job.getId());
        seedRequest.getQueryTemplateBuilder().getSeedBuilder().setDisabled(false);

        exe.submit(() -> produceSeedSubmissions(
                job,
                jobExecutionStatus,
                timeout,
                addToRunningJob,
                jobExecutionListeners,
                seedRequest.build(),
                jobExecutionStatusFuture));

        try {
            return jobExecutionStatusFuture.get();
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw Status.CANCELLED.withCause(e).asRuntimeException();
        } catch (ExecutionException e) {
            if (e.getCause() instanceof RuntimeException runtimeException) {
                throw runtimeException;
            }
            throw Status.UNKNOWN.withCause(e.getCause()).asRuntimeException();
        }
    }

    private static void produceSeedSubmissions(
            ConfigObject job,
            JobExecutionStatus existingJobExecutionStatus,
            OffsetDateTime timeout,
            boolean addToRunningJob,
            List<JobExecutionListener> jobExecutionListeners,
            ListRequest seedRequest,
            SettableFuture<JobExecutionStatus> jobExecutionStatusFuture) {
        AtomicLong submitted = new AtomicLong();
        AtomicLong processed = new AtomicLong();
        AtomicLong accepted = new AtomicLong();
        AtomicBoolean producerDone = new AtomicBoolean();
        AtomicBoolean jobExecutionStartedNotified = new AtomicBoolean();
        AtomicReference<String> lastFailure = new AtomicReference<>();
        Semaphore outstandingSubmissions = new Semaphore(MAX_OUTSTANDING_SEED_SUBMISSIONS);
        CompletionService<CrawlExecutionId> completionService =
                new ExecutorCompletionService<>(submitSeedExecutor, new LinkedBlockingQueue<>());

        boolean createdJobExecution = existingJobExecutionStatus == null;
        JobExecutionStatus currentJobExecutionStatus = existingJobExecutionStatus;
        boolean completionMonitorStarted = false;
        boolean jobExecutionStartingNotified = false;
        long enabledSeeds = 0L;
        long locallyRejectedSeeds = 0L;
        long duplicateSeeds = 0L;

        try (ChangeFeed<ConfigObject> seeds = DbService.getInstance().getConfigAdapter()
                .listConfigObjects(seedRequest)) {
            Iterator<ConfigObject> iterator = seeds.stream().iterator();

            while (iterator.hasNext()) {
                ConfigObject seed = iterator.next();
                enabledSeeds++;

                FrontierClient frontierClient = getFrontierClient(seed);
                if (frontierClient == null) {
                    locallyRejectedSeeds++;
                    lastFailure.set("No Frontier client is configured for seed '" + seed.getMeta().getName() + "'");
                    continue;
                }

                if (currentJobExecutionStatus != null && addToRunningJob
                        && isSeedInJobExecution(seed, currentJobExecutionStatus)) {
                    duplicateSeeds++;
                    continue;
                }

                // Preserve the existing validation of job script references, but do it only
                // after finding the first seed which can actually be submitted.
                if (!completionMonitorStarted) {
                    JobExecutionUtil.GetScriptAnnotationsForJob(job);
                    if (currentJobExecutionStatus == null) {
                        currentJobExecutionStatus = createJobExecutionStatusIfNotExist(job, null);
                    }
                    JobExecutionStatus monitoredJobExecutionStatus = currentJobExecutionStatus;
                    exe.execute(() -> monitorSeedSubmissions(
                            job,
                            monitoredJobExecutionStatus,
                            createdJobExecution,
                            completionService,
                            outstandingSubmissions,
                            submitted,
                            processed,
                            accepted,
                            producerDone,
                            jobExecutionStartedNotified,
                            lastFailure,
                            jobExecutionListeners));
                    completionMonitorStarted = true;
                }

                JobExecutionStatus activeJobExecutionStatus = currentJobExecutionStatus;
                if (activeJobExecutionStatus == null) {
                    throw new IllegalStateException("Job execution status was not initialized");
                }

                if (!jobExecutionStartingNotified) {
                    for (JobExecutionListener listener : jobExecutionListeners) {
                        listener.onJobStarting(activeJobExecutionStatus.getId());
                    }
                    jobExecutionStartingNotified = true;
                }

                outstandingSubmissions.acquire();
                submitted.incrementAndGet();
                try {
                    completionService.submit(() -> frontierClient.crawlSeed(
                            job, seed, activeJobExecutionStatus, timeout));
                    // Bulk RunCrawl acknowledges local submission and returns this provisional
                    // job execution ID without waiting for every Frontier call.
                    jobExecutionStatusFuture.set(activeJobExecutionStatus);
                } catch (RuntimeException e) {
                    submitted.decrementAndGet();
                    outstandingSubmissions.release();
                    locallyRejectedSeeds++;
                    lastFailure.set(e.toString());
                }
            }

            if (submitted.get() == 0L && !jobExecutionStatusFuture.isDone()) {
                if (duplicateSeeds > 0L && currentJobExecutionStatus != null) {
                    jobExecutionStatusFuture.set(currentJobExecutionStatus);
                } else if (enabledSeeds > 0L) {
                    jobExecutionStatusFuture.setException(Status.UNAVAILABLE
                            .withDescription("No Frontier client is configured for the enabled seeds in job '"
                                    + job.getMeta().getName() + "'")
                            .asRuntimeException());
                } else {
                    jobExecutionStatusFuture.set(null);
                    LOG.info("No enabled seeds are associated with job '{}'", job.getMeta().getName());
                }
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            lastFailure.set(e.toString());
            jobExecutionStatusFuture.setException(Status.CANCELLED
                    .withDescription("Interrupted while submitting seeds for a job execution")
                    .withCause(e)
                    .asRuntimeException());
        } catch (Exception e) {
            lastFailure.set(e.toString());
            jobExecutionStatusFuture.setException(e);
        } finally {
            producerDone.set(true);
            LOG.info("{} enabled seeds considered for job '{}'; {} submitted, {} locally rejected",
                    enabledSeeds, job.getMeta().getName(), submitted.get(), locallyRejectedSeeds);
        }
    }

    private static void monitorSeedSubmissions(
            ConfigObject job,
            JobExecutionStatus jobExecutionStatus,
            boolean createdJobExecution,
            CompletionService<CrawlExecutionId> completionService,
            Semaphore outstandingSubmissions,
            AtomicLong submitted,
            AtomicLong processed,
            AtomicLong accepted,
            AtomicBoolean producerDone,
            AtomicBoolean jobExecutionStartedNotified,
            AtomicReference<String> lastFailure,
            List<JobExecutionListener> jobExecutionListeners) {
        try {
            while (!producerDone.get() || processed.get() < submitted.get()) {
                Future<CrawlExecutionId> future = completionService.poll(
                        COMPLETION_POLL_MILLIS, TimeUnit.MILLISECONDS);
                if (future == null) {
                    continue;
                }

                processed.incrementAndGet();
                try {
                    CrawlExecutionId crawlExecutionId = future.get();
                    if (isAccepted(crawlExecutionId, null)) {
                        accepted.incrementAndGet();
                        LOG.trace("Crawl execution '{}' created for job execution '{}'",
                                crawlExecutionId.getId(), jobExecutionStatus.getId());
                    } else {
                        lastFailure.set("Frontier returned an empty crawl execution ID");
                    }
                } catch (ExecutionException e) {
                    lastFailure.set(e.getCause() == null ? e.toString() : e.getCause().toString());
                    LOG.info("Frontier rejected a seed submission for job execution '{}': {}",
                            jobExecutionStatus.getId(), lastFailure.get());
                } finally {
                    outstandingSubmissions.release();
                }
            }

            LOG.info("Frontier rejected {} of {} seed submissions for job '{}' and job execution '{}'",
                    submitted.get() - accepted.get(), submitted.get(),
                    job.getMeta().getName(), jobExecutionStatus.getId());

            if (accepted.get() == 0L && createdJobExecution) {
                JobExecutionStatus currentStatus = setJobExecutionStateFailedIfEmpty(
                        jobExecutionStatus, submitted.get(), lastFailure.get());
                if (currentStatus != null && currentStatus.getState() == JobExecutionStatus.State.RUNNING) {
                    // A crawl execution exists even though its Frontier response was lost.
                    notifyJobExecutionStarted(
                            jobExecutionStatus, jobExecutionStartedNotified, jobExecutionListeners);
                }
            } else if (accepted.get() > 0L) {
                notifyJobExecutionStarted(
                        jobExecutionStatus, jobExecutionStartedNotified, jobExecutionListeners);
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            LOG.warn("Interrupted while monitoring seed submissions for job execution '{}'",
                    jobExecutionStatus.getId(), e);
        } catch (DbException e) {
            LOG.error("Failed to update empty job execution '{}'", jobExecutionStatus.getId(), e);
        }
    }

    public static JobExecutionStatus setJobExecutionStateFailedIfEmpty(
            JobExecutionStatus jobExecutionStatus, long submitted, String lastFailure) throws DbException {
        String detail = "No crawl execution was created for job execution '" + jobExecutionStatus.getId()
                + "' after " + submitted + " seed submission(s).";
        if (lastFailure != null && !lastFailure.isBlank()) {
            detail += " Last failure: " + truncate(lastFailure, 512);
        }

        Error error = Error.newBuilder()
                .setCode(Status.Code.UNAVAILABLE.value())
                .setMsg("Frontier accepted no seed submissions")
                .setDetail(detail)
                .build();
        return DbService.getInstance().getExecutionsAdapter()
                .setJobExecutionStateFailedIfEmpty(jobExecutionStatus.getId(), error);
    }

    static boolean hasFrontierClient(ConfigObject seed) {
        return getFrontierClient(seed) != null;
    }

    private static FrontierClient getFrontierClient(ConfigObject seed) {
        String type = ApiTools.getFirstLabelWithKey(seed.getMeta(), SEED_TYPE_LABEL_KEY)
                .orElse(buildLabel(SEED_TYPE_LABEL_KEY, "url"))
                .getValue()
                .toLowerCase();
        return frontierClients.get(type);
    }

    public static boolean isSeedInJobExecution(ConfigObject seed, JobExecutionStatus jobExecutionStatus) {
        CrawlExecutionsListRequest.Builder request = CrawlExecutionsListRequest.newBuilder();
        request.getQueryTemplateBuilder()
                .setSeedId(seed.getId())
                .setJobExecutionId(jobExecutionStatus.getId());
        request.getQueryMaskBuilder()
                .addPaths("seedId")
                .addPaths("jobExecutionId");
        try (ChangeFeed<?> crawlExecutions = DbService.getInstance().getExecutionsAdapter()
                .listCrawlExecutionStatus(request.build())) {
            return crawlExecutions.stream().findAny().isPresent();
        } catch (DbException e) {
            LOG.warn("Could not check whether seed '{}' already has a crawl execution in job execution '{}'",
                    seed.getMeta().getName(), jobExecutionStatus.getId(), e);
            return false;
        }
    }

    private static boolean isAccepted(CrawlExecutionId crawlExecutionId, ConfigObject seed) {
        if (crawlExecutionId != null && !crawlExecutionId.getId().isEmpty()) {
            return true;
        }
        if (seed != null) {
            LOG.warn("Frontier returned an empty crawl execution ID for seed '{}'", seed.getMeta().getName());
        }
        return false;
    }

    private static void notifyJobExecutionStarted(
            JobExecutionStatus jobExecutionStatus,
            AtomicBoolean jobExecutionStartedNotified,
            List<JobExecutionListener> jobExecutionListeners) {
        if (jobExecutionStartedNotified.compareAndSet(false, true)) {
            for (JobExecutionListener listener : jobExecutionListeners) {
                listener.onJobStarted(jobExecutionStatus.getId());
            }
        }
    }

    private static String truncate(String value, int maxLength) {
        return value.length() <= maxLength ? value : value.substring(0, maxLength);
    }

    /**
     * Gets a running jobExecution for a job or null if not running.
     *
     * @param job the job to check if running
     * @return a JobExecutionStatus with state RUNNING or null if no job was not running
     * @throws DbException
     */
    public static JobExecutionStatus getRunningJobExecutionStatusForJob(ConfigObject job) throws DbException {
        ChangeFeed<JobExecutionStatus> runningJobsR = DbService.getInstance().getExecutionsAdapter()
                .listJobExecutionStatus(JobExecutionsListRequest.newBuilder()
                        .setQueryMask(FieldMask.newBuilder().addPaths("jobId"))
                        .setQueryTemplate(JobExecutionStatus.newBuilder().setJobId(job.getId()))
                        .addState(State.RUNNING).build());

        return runningJobsR.stream().findAny().orElse(null);
    }

    public static JobExecutionStatus createJobExecutionStatusIfNotExist(ConfigObject job, JobExecutionStatus existingJobExecutionStatus) throws DbException {
        if (existingJobExecutionStatus == null) {
            JobExecutionStatus jes = DbService.getInstance().getExecutionsAdapter()
                    .createJobExecutionStatus(job.getId());
            LOG.info("Creating new job execution '{}'", jes.getId());
            return jes;
        } else {
            return existingJobExecutionStatus;
        }
    }

    public static void queueCountAndBusyChgCount(FutureCallback<CrawlerStatus.Builder> callback) {
        String type = "url";
        FrontierClient frontierClient = frontierClients.get(type);

        if (frontierClient != null) {
            ListenableFuture<CountResponse> queueCount = frontierClient.queueCountTotal();
            ListenableFuture<CountResponse> chgCount = frontierClient.busyCrawlHostGroupCount();
            ListenableFuture<CrawlerStatus.Builder> queueAndChgCount = Futures.whenAllSucceed(queueCount, chgCount)
                    .call(
                            () -> CrawlerStatus.newBuilder()
                                    .setBusyCrawlHostGroupCount(Futures.getDone(chgCount).getCount())
                                    .setQueueSize(Futures.getDone(queueCount).getCount()),
                            exe);
            Futures.addCallback(queueAndChgCount, callback, exe);
        } else {
            LOG.warn("No frontier defined for seed type {}", type);
            callback.onFailure(new IllegalArgumentException("No frontier defined for seed type " + type));
        }
    }

    public static void queueCountsForCrawlExecutions(
            ExecutionIds executionIds, FutureCallback<QueueCountsResponse> callback) {
        FrontierClient frontierClient = frontierClients.get("url");
        if (frontierClient != null) {
            frontierClient.queueCountsForCrawlExecutions(executionIds, callback, exe);
        } else {
            LOG.warn("No frontier defined for seed type url");
            callback.onFailure(new IllegalArgumentException("No frontier defined for seed type url"));
        }
    }

    public static void queueCountsForJobExecutions(
            ExecutionIds executionIds, FutureCallback<QueueCountsResponse> callback) {
        FrontierClient frontierClient = frontierClients.get("url");
        if (frontierClient != null) {
            frontierClient.queueCountsForJobExecutions(executionIds, callback, exe);
        } else {
            LOG.warn("No frontier defined for seed type url");
            callback.onFailure(new IllegalArgumentException("No frontier defined for seed type url"));
        }
    }

    public static void queueCountForCrawlHostGroup(CrawlHostGroup crawlHostGroup, FutureCallback<CountResponse> callback) {
        String type = "url";
        FrontierClient frontierClient = frontierClients.get(type);

        if (frontierClient != null) {
            frontierClient.queueCountForCrawlHostGroup(crawlHostGroup, callback, exe);
        } else {
            LOG.warn("No frontier defined for seed type {}", type);
            callback.onFailure(new IllegalArgumentException("No frontier defined for seed type " + type));
        }
    }

    public static OffsetDateTime calculateTimeout(ConfigObject job) {
        OffsetDateTime timeout = null;
        if (job.getCrawlJob().hasLimits()) {
            long maxDurationS = job.getCrawlJob().getLimits().getMaxDurationS();
            if (maxDurationS > 0) {
                timeout = OffsetDateTime.now().plus(maxDurationS, ChronoUnit.SECONDS);
            }
        }
        return timeout;
    }

    public static Map<String, Annotation> GetScriptAnnotationsForJob(ConfigObject jobConfig) throws DbException {
        ConfigAdapter db = DbService.getInstance().getConfigAdapter();

        ConfigObject crawlConfig = db.getConfigObject(jobConfig.getCrawlJob().getCrawlConfigRef());
        ConfigObject browserConfig = db.getConfigObject(crawlConfig.getCrawlConfig().getBrowserConfigRef());

        Map<String, Annotation> annotations = new HashMap<>();

        // Get scope script annotations
        if (!jobConfig.getCrawlJob().hasScopeScriptRef()) {
            throw new IllegalArgumentException("Missing scopescript ref for crawl job " + jobConfig.getId());
        }
        db.getConfigObject(jobConfig.getCrawlJob().getScopeScriptRef()).getMeta().getAnnotationList()
                .forEach(a -> annotations.put(a.getKey(), a));

        // Get annotations for referenced browser scripts
        browserConfig.getBrowserConfig().getScriptRefList().forEach(r -> {
            try {
                db.getConfigObject(r).getMeta().getAnnotationList().forEach(a -> annotations.put(a.getKey(), a));
            } catch (DbException e) {
                throw new RuntimeException(e);
            }
        });

        // Get annotations for browser scripts matching selectors
        db.listConfigObjects(ListRequest.newBuilder().setKind(Kind.browserScript).addAllLabelSelector(
                browserConfig.getBrowserConfig().getScriptSelectorList()).build()).stream()
                .flatMap(s -> s.getMeta().getAnnotationList().stream())
                .forEach(a -> annotations.put(a.getKey(), a));

        // Override with job specific annotations
        jobConfig.getMeta().getAnnotationList().stream()
                .filter(a -> annotations.containsKey(a.getKey()))
                .forEach(a -> annotations.put(a.getKey(), a));

        return Collections.unmodifiableMap(annotations);
    }

    public static Map<String, Annotation> GetScriptAnnotationOverridesForSeed(
            ConfigObject seed, ConfigObject jobConfig, Map<String, Annotation> annotations) throws DbException {

        ConfigAdapter db = DbService.getInstance().getConfigAdapter();

        Map<String, Annotation> result = new HashMap<>();
        result.putAll(annotations);

        if (seed.getSeed().hasEntityRef()) {
            overrideAnnotation(db.getConfigObject(seed.getSeed().getEntityRef()).getMeta().getAnnotationList(), jobConfig, result);
        }

        overrideAnnotation(seed.getMeta().getAnnotationList(), jobConfig, result);

        return result;
    }

    static void overrideAnnotation(List<Annotation> annotations, ConfigObject jobConfig, Map<String, Annotation> jobAnnotations) {
        List<Annotation> ann = new ArrayList<>();
        ann.addAll(annotations);
        for (Iterator<Annotation> it = ann.iterator(); it.hasNext(); ) {
            Annotation a = it.next();
            if (jobAnnotations.containsKey(a.getKey())) {
                jobAnnotations.put(a.getKey(), a);
                it.remove();
            }
        }
        for (Annotation a : ann) {
            if (a.getKey().startsWith("{")) {
                int endIdx = a.getKey().indexOf('}');
                if (endIdx == -1) {
                    throw new IllegalArgumentException("Missing matching '}' for annotation: " + a.getKey());
                }
                String jobIdOrName = a.getKey().substring(1, endIdx);
                String key = a.getKey().substring(endIdx + 1);
                if ((jobConfig.getId().equals(jobIdOrName) || jobConfig.getMeta().getName().equals(jobIdOrName))
                        && jobAnnotations.containsKey(key)) {
                    a = a.toBuilder().setKey(key).build();
                    jobAnnotations.put(a.getKey(), a);
                }
            }
        }
    }

    @FunctionalInterface
    public interface CheckedSupplier<T, E extends Exception> {
        T get() throws E;
    }
}
