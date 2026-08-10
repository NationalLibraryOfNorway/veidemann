import {ComponentFixture, TestBed} from '@angular/core/testing';

import {AbortCrawlDialogComponent} from './abort-crawl-dialog.component';
import {MAT_DIALOG_DATA, MatDialog, MatDialogRef} from '@angular/material/dialog';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {CrawlExecutionStatus, JobExecutionStatus} from '../../../../shared/models';

describe('AbortCrawlDialogComponent', () => {
  let component: AbortCrawlDialogComponent;
  let fixture: ComponentFixture<AbortCrawlDialogComponent>;
  let close: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    close = vi.fn();
    TestBed.configureTestingModule({
      imports: [AbortCrawlDialogComponent],
      providers: [
        ...provideCoreTesting,
        {
          provide: MAT_DIALOG_DATA,
          useValue: {jobExecutionStatus: new JobExecutionStatus({id: 'job-execution-1'})},
        },
        {provide: MatDialogRef, useValue: {close}}
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(AbortCrawlDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('shows concise retention copy and places cancel before the destructive confirmation', () => {
    const title = fixture.nativeElement.querySelector('[mat-dialog-title]') as HTMLElement;
    const content = fixture.nativeElement.querySelector('[mat-dialog-content]') as HTMLElement;
    const buttons = [
      ...fixture.nativeElement.querySelectorAll('[mat-dialog-actions] button')
    ] as HTMLButtonElement[];

    expect(title.textContent.trim()).toBe('Abort crawl?');
    expect(content.textContent.replace(/\s+/g, ' ').trim()).toBe(
      'The crawl will stop processing new URLs. Already crawled documents will be retained.'
    );
    expect(buttons.map(button => button.textContent.trim())).toEqual(['Cancel', 'Abort crawl']);
    expect(buttons[1].classList).toContain('confirm-abort');
  });

  it('closes with the execution id only after confirmation', () => {
    const buttons = fixture.nativeElement.querySelectorAll(
      '[mat-dialog-actions] button'
    ) as NodeListOf<HTMLButtonElement>;

    buttons[1].click();

    expect(close).toHaveBeenCalledOnce();
    expect(close.mock.calls[0][0].id).toBe('job-execution-1');
  });

  it('returns a crawl execution id when aborting one crawl execution', () => {
    component.data = {
      crawlExecutionStatus: new CrawlExecutionStatus({id: 'crawl-execution-1'}),
    };

    component.onAbortCrawl();

    expect(close.mock.calls[0][0].id).toBe('crawl-execution-1');
  });

  it('closes with false when cancel is selected', () => {
    const cancel = fixture.nativeElement.querySelector(
      '[mat-dialog-actions] button'
    ) as HTMLButtonElement;

    cancel.click();

    expect(close).toHaveBeenCalledWith(false);
  });

  it('closes without confirming when Escape is pressed', async () => {
    const dialogRef = TestBed.inject(MatDialog).open(AbortCrawlDialogComponent, {
      data: {},
      disableClose: false,
    });
    let closed = false;
    dialogRef.afterClosed().subscribe(() => closed = true);

    const escapeEvent = new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
      cancelable: true,
    });
    Object.defineProperty(escapeEvent, 'keyCode', {get: () => 27});
    document.body.dispatchEvent(escapeEvent);
    await fixture.whenStable();

    expect(closed).toBe(true);
  });
});
