import {TestBed} from '@angular/core/testing';
import {EMPTY, firstValueFrom, of} from 'rxjs';

import {ConfigObject, CrawlJob, Kind} from '../../../shared/models';
import {JobExecutionService} from '../services';
import {JobMaxDurationPipe} from './job-max-duration.pipe';

describe('JobMaxDurationPipe', () => {
  let getJob: ReturnType<typeof vi.fn>;
  let pipe: JobMaxDurationPipe;

  beforeEach(() => {
    getJob = vi.fn();
    TestBed.configureTestingModule({
      providers: [{provide: JobExecutionService, useValue: {getJob}}],
    });
    pipe = TestBed.runInInjectionContext(() => new JobMaxDurationPipe());
  });

  it('returns the maximum duration of the job in seconds', async () => {
    getJob.mockReturnValue(of(new ConfigObject({
      kind: Kind.CRAWLJOB,
      crawlJob: new CrawlJob({limits: {maxDurationS: 7200, maxBytes: 0}}),
    })));

    expect(await firstValueFrom(pipe.transform('job-1'))).toBe(7200);
    expect(getJob).toHaveBeenCalledWith('job-1');
  });

  it('returns 0 when the job has no limit, no crawl job spec or cannot be loaded', async () => {
    getJob.mockReturnValueOnce(of(new ConfigObject({kind: Kind.CRAWLJOB})));
    expect(await firstValueFrom(pipe.transform('no-spec'))).toBe(0);

    getJob.mockReturnValueOnce(EMPTY);
    expect(await firstValueFrom(pipe.transform('unavailable'))).toBe(0);
  });
});
