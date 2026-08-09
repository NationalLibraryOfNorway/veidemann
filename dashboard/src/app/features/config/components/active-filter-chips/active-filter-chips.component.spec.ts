import {ComponentFixture, TestBed} from '@angular/core/testing';

import {provideMaterialAnimationsDisabled} from '../../../../core/core.testing.module';
import {ConfigQuery} from '../../../../shared/func';
import {BrowserScriptType, ConfigObject, Kind, Meta} from '../../../../shared/models';
import {ActiveConfigFilterChip, ActiveFilterChipsComponent} from './active-filter-chips.component';

describe('ActiveFilterChipsComponent', () => {
  let fixture: ComponentFixture<ActiveFilterChipsComponent>;

  const query: ConfigQuery = {
    kind: Kind.SEED,
    entityId: 'entity-1',
    scheduleId: 'schedule-1',
    crawlConfigId: 'crawl-config-1',
    collectionId: 'collection-1',
    browserConfigId: 'browser-config-1',
    politenessId: 'politeness-1',
    disabled: false,
    browserScriptType: BrowserScriptType.ON_LOAD,
    robotsPolicy: null,
    role: null,
    crawlJobIdList: ['job-1', 'job-missing'],
    scriptIdList: ['script-1'],
    term: 'search text',
    active: 'meta.name',
    direction: 'asc',
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ActiveFilterChipsComponent],
      providers: [provideMaterialAnimationsDisabled()],
    }).compileComponents();

    fixture = TestBed.createComponent(ActiveFilterChipsComponent);
    fixture.componentRef.setInput('query', query);
    fixture.componentRef.setInput('entity', new ConfigObject({
      id: 'entity-1',
      kind: Kind.CRAWLENTITY,
      meta: new Meta({name: 'Example entity'}),
    }));
    fixture.componentRef.setInput('options', {
      crawlJobs: [new ConfigObject({id: 'job-1', kind: Kind.CRAWLJOB, meta: new Meta({name: 'Daily job'})})],
      browserScripts: [new ConfigObject({
        id: 'script-1',
        kind: Kind.BROWSERSCRIPT,
        meta: new Meta({name: 'Scope script'}),
      })],
    });
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('shows named Entity and BrowserScript filters without duplicating CrawlJobs', () => {
    const text = fixture.nativeElement.textContent;

    expect(text).toContain('Entity: Example entity');
    expect(text).toContain('BrowserScript: Scope script');
    expect(text).not.toContain('Crawljob:');
    expect(text).not.toContain('Daily job');
    expect(text).not.toContain('job-missing');
  });

  it('does not duplicate single-select, status, or search controls as chips', () => {
    const text = fixture.nativeElement.textContent;

    expect(text).not.toContain('schedule-1');
    expect(text).not.toContain('crawl-config-1');
    expect(text).not.toContain('False');
    expect(text).not.toContain('ON_LOAD');
    expect(text).not.toContain('search text');
  });

  it('emits only the removed chip', () => {
    let removed: ActiveConfigFilterChip | undefined;
    fixture.componentInstance.removeFilter.subscribe(chip => removed = chip);

    const removeButtons = fixture.nativeElement.querySelectorAll('button[matChipRemove]') as NodeListOf<HTMLButtonElement>;
    removeButtons[1].click();

    expect(removed).toEqual(expect.objectContaining({key: 'scriptIdList', value: 'script-1'}));
  });

  it('renders ordinary and key-only applied label filters', async () => {
    fixture.componentRef.setInput('query', {...query, term: 'example label:owner:archive'});
    fixture.detectChanges();
    await fixture.whenStable();

    let labelChip = [...fixture.nativeElement.querySelectorAll('mat-chip')]
      .find((chip: HTMLElement) => chip.textContent.includes('owner:archive')) as HTMLElement;
    expect(labelChip).toBeTruthy();

    fixture.componentRef.setInput('query', {...query, term: 'label:owner'});
    fixture.detectChanges();
    await fixture.whenStable();
    labelChip = [...fixture.nativeElement.querySelectorAll('mat-chip')]
      .find((chip: HTMLElement) => chip.textContent.replace('cancel', '').trim() === 'owner') as HTMLElement;
    expect(labelChip).toBeTruthy();
  });

  it('renders an emoji label visually with accessible text', async () => {
    fixture.componentRef.setInput('query', {...query, term: 'label:emoji:🐶'});
    fixture.detectChanges();
    await fixture.whenStable();

    const labelChip = [...fixture.nativeElement.querySelectorAll('mat-chip')]
      .find((chip: HTMLElement) => !!chip.querySelector('app-label-display')) as HTMLElement;
    expect(labelChip.querySelector('.label-display__emoji')?.textContent).toBe('🐶');
    expect(labelChip.querySelector('.label-display__accessible')?.textContent).toBe('emoji:🐶');
  });

  it('emits the applied label selector when its chip is removed', async () => {
    let removed: ActiveConfigFilterChip | undefined;
    fixture.componentInstance.removeFilter.subscribe(chip => removed = chip);
    fixture.componentRef.setInput('query', {...query, term: 'label:owner:archive'});
    fixture.detectChanges();
    await fixture.whenStable();

    const remove = [...fixture.nativeElement.querySelectorAll('button[matChipRemove]')]
      .find((button: HTMLButtonElement) => button.getAttribute('aria-label') === 'Remove owner:archive label filter');
    (remove as HTMLButtonElement).click();

    expect(removed).toEqual(expect.objectContaining({
      key: 'labelSelector',
      value: 'owner:archive',
    }));
  });
});
