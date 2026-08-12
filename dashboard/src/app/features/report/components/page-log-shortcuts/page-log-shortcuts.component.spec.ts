import {Clipboard} from '@angular/cdk/clipboard';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {PageLogShortcutsComponent} from './page-log-shortcuts.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {SnackBarService} from '../../../../core';
import {PageLog} from '../../../../shared/models';

describe('PageLogShortcutsComponent', () => {
  let component: PageLogShortcutsComponent;
  let fixture: ComponentFixture<PageLogShortcutsComponent>;
  let can: ReturnType<typeof vi.fn>;
  let copy: ReturnType<typeof vi.fn>;
  let openSnackBar: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    can = vi.fn(() => true);
    copy = vi.fn(() => true);
    openSnackBar = vi.fn();
    await TestBed.configureTestingModule({
      imports: [PageLogShortcutsComponent],
      providers:[
        ...provideCoreTesting,
        provideRouter([]),
        {provide: AbilityServiceSignal, useValue: {can}},
        {provide: Clipboard, useValue: {copy}},
        {provide: SnackBarService, useValue: {openSnackBar, openError: vi.fn()}},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(PageLogShortcutsComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('pageLog', new PageLog());
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('renders header actions while leaving method and collection to the heading', () => {
    fixture.componentRef.setInput('pageLog', new PageLog({
      warcId: 'warc-1',
      referrer: 'https://referrer.example/path',
      collectionFinalName: 'collection-1',
      method: 'GET',
      jobExecutionId: 'job-execution-1',
      executionId: 'crawl-execution-1',
    }));
    fixture.detectChanges();

    const chips = fixture.nativeElement.querySelectorAll('mat-chip, a[mat-chip]') as NodeListOf<HTMLElement>;
    const links = fixture.nativeElement.querySelectorAll('a[mat-chip]') as NodeListOf<HTMLAnchorElement>;
    const copyChip = fixture.nativeElement.querySelector('[aria-label="Copy WARC ID"]') as HTMLElement;
    expect(chips.length).toBe(4);
    expect(fixture.nativeElement.textContent).toContain('Copy WARC ID');
    expect(fixture.nativeElement.textContent).not.toContain('collection-1');
    expect(fixture.nativeElement.textContent).not.toContain('GET');
    expect(getComputedStyle(copyChip).cursor).toBe('pointer');
    expect([...links].every(link => getComputedStyle(link).cursor === 'pointer')).toBe(true);
    expect(links[0].href).toBe('https://referrer.example/path');
    expect([...links].map(link => link.getAttribute('href'))).toEqual(expect.arrayContaining([
      '/report/jobexecution/job-execution-1',
      '/report/crawlexecution/crawl-execution-1',
    ]));

    copyChip.click();
    expect(copy).toHaveBeenCalledWith('warc-1');
    expect(openSnackBar).toHaveBeenCalledWith('ID copied');
  });

  it('keeps invalid referrers and unauthorized execution IDs as plain chips and omits empty metadata', () => {
    can.mockReturnValue(false);
    fixture.componentRef.setInput('pageLog', new PageLog({
      warcId: 'warc-1',
      referrer: 'not a URI',
      jobExecutionId: 'job-execution-1',
      executionId: 'crawl-execution-1',
    }));
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Referrer: not a URI');
    expect(fixture.nativeElement.textContent).not.toContain('Collection:');
    expect(fixture.nativeElement.textContent).not.toContain('Method:');
    expect(fixture.nativeElement.querySelectorAll('a[mat-chip]').length).toBe(0);
    expect(fixture.nativeElement.textContent).toContain('Job execution');
    expect(fixture.nativeElement.textContent).toContain('Crawl execution');
    expect(fixture.nativeElement.textContent).not.toContain('job-execution-1');
  });

  it('renders the same destinations and copy command as menu items', () => {
    fixture.componentRef.setInput('presentation', 'menu');
    fixture.componentRef.setInput('pageLog', new PageLog({
      warcId: 'warc-1',
      referrer: 'https://referrer.example/path',
      jobExecutionId: 'job-execution-1',
      executionId: 'crawl-execution-1',
    }));
    fixture.detectChanges();

    const links = [...fixture.nativeElement.querySelectorAll('a[mat-menu-item]')] as HTMLAnchorElement[];
    expect(links.map(link => link.getAttribute('href'))).toEqual([
      'https://referrer.example/path',
      '/report/jobexecution/job-execution-1',
      '/report/crawlexecution/crawl-execution-1',
    ]);
    expect(fixture.nativeElement.querySelectorAll('button[mat-menu-item]').length).toBe(1);
    expect(fixture.nativeElement.querySelector('mat-chip-set')).toBeNull();
  });
});
