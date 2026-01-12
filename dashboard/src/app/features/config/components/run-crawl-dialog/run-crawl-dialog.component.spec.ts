import {TestBed} from '@angular/core/testing';
import {By} from '@angular/platform-browser';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';

import {MatButtonHarness} from '@angular/material/button/testing';
import {MatSelectHarness} from '@angular/material/select/testing';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';

import {RunCrawlDialogComponent} from './run-crawl-dialog.component';
import {ConfigObject, CrawlJob, Kind, Meta} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';

/* --------------------------------------------------------------------
 * Test data
 * ------------------------------------------------------------------ */

const exampleCrawljobs = [
  {
    id: 'configObject_id',
    apiVersion: 'v1',
    kind: Kind.CRAWLJOB,
    meta: new Meta({ name: 'Example CrawlJob' }),
    crawlJob: new CrawlJob({}),
    disabled: false,
  },
  {
    id: 'configObject_id2',
    apiVersion: 'v1',
    kind: Kind.CRAWLJOB,
    meta: new Meta({ name: 'Example CrawlJob2' }),
    crawlJob: new CrawlJob({}),
    disabled: false,
  },
];

const exampleCrawljobToCrawl = {
  runCrawlReply: { jobExecutionId: 'testid' },
  configObject: new ConfigObject({
    id: 'test_crawljob_id',
    apiVersion: 'v1',
    kind: Kind.CRAWLJOB,
    meta: new Meta({ name: 'Example Crawljob' }),
    crawlJob: new CrawlJob({}),
  }),
  crawlJobs: exampleCrawljobs,
};

const exampleSeedToCrawl = {
  runCrawlReply: { jobExecutionId: 'testid' },
  configObject: new ConfigObject({
    id: 'test_seed_id',
    apiVersion: 'v1',
    kind: Kind.SEED,
    meta: new Meta({ name: 'https://www.nb.no' }),
  }),
  crawlJobs: exampleCrawljobs,
};

const exampleSeedsToCrawl = {
  ...exampleSeedToCrawl,
  numberOfSeeds: 3,
};

/* --------------------------------------------------------------------
 * Helper
 * ------------------------------------------------------------------ */

async function setup(data: any) {
  const dialogRefMock = {
    close: vi.fn(),
  };

  await TestBed.configureTestingModule({
    imports: [RunCrawlDialogComponent],
    providers: [
      ...provideCoreTesting,
      { provide: MAT_DIALOG_DATA, useValue: data },
      { provide: MatDialogRef, useValue: dialogRefMock },
    ],
  });

  const fixture = TestBed.createComponent(RunCrawlDialogComponent);
  const component = fixture.componentInstance;
  const loader = TestbedHarnessEnvironment.loader(fixture);

  fixture.detectChanges();
  await fixture.whenStable();

  return { fixture, component, loader, dialogRefMock };
}

/* --------------------------------------------------------------------
 * Tests
 * ------------------------------------------------------------------ */

describe('RunCrawlDialogComponent', () => {

  describe('Crawljob run', () => {
    it('should display correct crawljob confirmation text', async () => {
      const { fixture } = await setup(exampleCrawljobToCrawl);

      fixture.detectChanges();
      await fixture.whenStable();
      fixture.detectChanges();

      const el = fixture.debugElement.query(
        By.css('[data-testid="run_crawljob_dialog_text"]')
      );

      expect(el).not.toBeNull();

      const text = el!.nativeElement.textContent?.trim();
      expect(text).toContain('Example Crawljob');
    });

    it('should close dialog with RunCrawlRequest when RUN clicked', async () => {
      const { loader, dialogRefMock } = await setup(exampleCrawljobToCrawl);

      const runButton = await loader.getHarness(
        MatButtonHarness.with({ text: 'RUN' })
      );

      await runButton.click();

      expect(dialogRefMock.close).toHaveBeenCalledTimes(1);
    });
  });

  describe('Single seed run', () => {
    it('should display correct seed confirmation text', async () => {
      const { fixture } = await setup(exampleSeedToCrawl);

      fixture.detectChanges();
      await fixture.whenStable();
      fixture.detectChanges();

      const el = fixture.debugElement.query(
        By.css('[data-testid="run_seed_dialog_text"]')
      );

      expect(el).not.toBeNull();

      const text = el!.nativeElement.textContent?.trim();
      expect(text).toContain('https://www.nb.no');
    });

    it('should enable RUN button after crawljob selection', async () => {
      const { loader, component, fixture } = await setup(exampleSeedToCrawl);

      const select = await loader.getHarness(MatSelectHarness);
      await select.open();

      const options = await select.getOptions();
      await options[0].click();

      fixture.detectChanges();

      expect(component.jobRefId).toBe('configObject_id');

      const runButton = await loader.getHarness(
        MatButtonHarness.with({ text: 'RUN' })
      );

      expect(await runButton.isDisabled()).toBe(false);
    });

    it('should close dialog when RUN clicked', async () => {
      const { loader, dialogRefMock, fixture } = await setup(exampleSeedToCrawl);

      const select = await loader.getHarness(MatSelectHarness);
      await select.open();
      const options = await select.getOptions();
      await options[0].click();

      fixture.detectChanges();

      const runButton = await loader.getHarness(
        MatButtonHarness.with({ text: 'RUN' })
      );

      await runButton.click();

      expect(dialogRefMock.close).toHaveBeenCalledTimes(1);
    });
  });

  describe('Multiple seeds run', () => {
    it('should display correct number of seeds', async () => {
      const { fixture } = await setup(exampleSeedsToCrawl);

      fixture.detectChanges();
      await fixture.whenStable();
      fixture.detectChanges();

      const el = fixture.debugElement.query(
        By.css('[data-testid="run_multiple_seeds_dialog_text"]')
      );

      expect(el).not.toBeNull();

      const text = el!.nativeElement.textContent?.trim();
      expect(text).toContain('3');
    });

    it('should list all crawljob options', async () => {
      const { loader } = await setup(exampleSeedsToCrawl);

      const select = await loader.getHarness(MatSelectHarness);
      await select.open();

      const options = await select.getOptions();
      expect(options.length).toBe(2);
    });

    it('should enable RUN button after crawljob selection', async () => {
      const { loader, component, fixture } = await setup(exampleSeedsToCrawl);

      const select = await loader.getHarness(MatSelectHarness);
      await select.open();
      const options = await select.getOptions();
      await options[0].click();

      fixture.detectChanges();

      expect(component.jobRefId).toBe('configObject_id');

      const runButton = await loader.getHarness(
        MatButtonHarness.with({ text: 'RUN' })
      );

      expect(await runButton.isDisabled()).toBe(false);
    });
  });
});
