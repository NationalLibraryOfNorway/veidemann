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
});
