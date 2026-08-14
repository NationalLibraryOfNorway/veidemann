import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatTooltip} from '@angular/material/tooltip';
import {By} from '@angular/platform-browser';

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

  it('shows Crawl Job and BrowserScript filters with kind icons and name fallbacks', () => {
    const text = fixture.nativeElement.textContent;
    const dailyJobChip = [...fixture.nativeElement.querySelectorAll('mat-chip')]
      .find((chip: HTMLElement) => chip.textContent.includes('Daily job')) as HTMLElement;
    const missingJobChip = [...fixture.nativeElement.querySelectorAll('mat-chip')]
      .find((chip: HTMLElement) => chip.textContent.includes('job-missing')) as HTMLElement;
    const scriptChip = [...fixture.nativeElement.querySelectorAll('mat-chip')]
      .find((chip: HTMLElement) => chip.textContent.includes('Scope script')) as HTMLElement;

    expect(text).toContain('Example entity');
    expect(text).not.toContain('Entity: Example entity');
    expect(dailyJobChip.querySelector('mat-icon[matChipAvatar]')?.textContent.trim()).toBe('work');
    expect(missingJobChip.querySelector('mat-icon[matChipAvatar]')?.textContent.trim()).toBe('work');
    expect(dailyJobChip.querySelector('button[matChipRemove]')?.getAttribute('aria-label'))
      .toBe('Remove Daily job crawl job filter');
    expect(text).toContain('Scope script');
    expect(text).not.toContain('BrowserScript:');
    expect(scriptChip.querySelector('mat-icon[matChipAvatar]')?.textContent.trim()).toBe('web_asset');
    expect(scriptChip.querySelector('button[matChipRemove]')?.getAttribute('aria-label'))
      .toBe('Remove BrowserScript Scope script filter');
    expect(text).not.toContain('Crawljob:');
    const entityChip = [...fixture.nativeElement.querySelectorAll('mat-chip')]
      .find((chip: HTMLElement) => chip.textContent.includes('Example entity')) as HTMLElement;
    expect(entityChip.querySelector('mat-icon[matChipAvatar]')?.textContent.trim()).toBe('business');
    const entityChipDebug = fixture.debugElement.queryAll(By.css('mat-chip'))
      .find(element => element.nativeElement === entityChip);
    expect(entityChipDebug?.injector.get(MatTooltip).message).toBe('Entity ID: entity-1');
    expect(entityChip.querySelector('button[matChipRemove]')?.getAttribute('aria-label'))
      .toBe('Remove entity entity-1 filter');
  });

  it('does not duplicate single-select, status, or search controls as chips', () => {
    const text = fixture.nativeElement.textContent;

    expect(text).not.toContain('schedule-1');
    expect(text).not.toContain('crawl-config-1');
    expect(text).not.toContain('False');
    expect(text).not.toContain('ON_LOAD');
    expect(text).not.toContain('search text');
  });

  it('emits only the selected BrowserScript chip', () => {
    let removed: ActiveConfigFilterChip | undefined;
    fixture.componentInstance.removeFilter.subscribe(chip => removed = chip);

    const scriptChip = [...fixture.nativeElement.querySelectorAll('mat-chip')]
      .find((chip: HTMLElement) => chip.textContent.includes('Scope script')) as HTMLElement;
    (scriptChip.querySelector('button[matChipRemove]') as HTMLButtonElement).click();

    expect(removed).toEqual(expect.objectContaining({key: 'scriptIdList', value: 'script-1'}));
  });

  it('disables filter chips and prevents their removal', () => {
    let removed: ActiveConfigFilterChip | undefined;
    fixture.componentInstance.removeFilter.subscribe(chip => removed = chip);
    fixture.componentRef.setInput('disabled', true);
    fixture.detectChanges();

    const chips = [...fixture.nativeElement.querySelectorAll('mat-chip')] as HTMLElement[];
    const removeButtons = [
      ...fixture.nativeElement.querySelectorAll('button[matChipRemove]'),
    ] as HTMLButtonElement[];
    expect(chips.every(chip => chip.classList.contains('mat-mdc-chip-disabled'))).toBe(true);
    expect(removeButtons.every(button => button.disabled)).toBe(true);

    fixture.componentInstance.onRemoveFilter(fixture.componentInstance.chips[0]);
    expect(removed).toBeUndefined();
  });

  it('orders an ordinary or key-only label first and gives it the label icon', async () => {
    fixture.componentRef.setInput('query', {...query, term: 'example label:owner:archive'});
    fixture.detectChanges();
    await fixture.whenStable();

    let labelChip = [...fixture.nativeElement.querySelectorAll('mat-chip')]
      .find((chip: HTMLElement) => chip.textContent.includes('owner:archive')) as HTMLElement;
    expect(fixture.componentInstance.chips.map(chip => chip.key)).toEqual([
      'labelSelector',
      'crawlJobIdList',
      'crawlJobIdList',
      'entityId',
      'scriptIdList',
    ]);
    expect(labelChip.querySelector('mat-icon[matChipAvatar]')?.textContent.trim()).toBe('label');

    fixture.componentRef.setInput('query', {...query, term: 'label:owner'});
    fixture.detectChanges();
    await fixture.whenStable();
    labelChip = [...fixture.nativeElement.querySelectorAll('mat-chip')]
      .find((chip: HTMLElement) => chip.querySelector('button[matChipRemove]')
        ?.getAttribute('aria-label') === 'Remove owner label filter') as HTMLElement;
    expect(labelChip.querySelector('mat-icon[matChipAvatar]')?.textContent.trim()).toBe('label');
  });

  it('renders an emoji label visually with accessible text', async () => {
    fixture.componentRef.setInput('query', {...query, term: 'label:emoji:🐶'});
    fixture.detectChanges();
    await fixture.whenStable();

    const labelChip = [...fixture.nativeElement.querySelectorAll('mat-chip')]
      .find((chip: HTMLElement) => !!chip.querySelector('app-label-display')) as HTMLElement;
    expect(fixture.componentInstance.chips[0].key).toBe('labelSelector');
    expect(labelChip.querySelector('mat-icon[matChipAvatar]')).toBeNull();
    expect(labelChip.querySelector('.label-display__emoji')?.textContent).toBe('🐶');
    expect(labelChip.querySelector('.label-display__accessible')?.textContent).toBe('emoji:🐶');
  });

  it('emits one Crawl Job filter for removal', () => {
    let removed: ActiveConfigFilterChip | undefined;
    fixture.componentInstance.removeFilter.subscribe(chip => removed = chip);
    const crawlJobChip = [...fixture.nativeElement.querySelectorAll('mat-chip')]
      .find((chip: HTMLElement) => chip.textContent.includes('Daily job')) as HTMLElement;

    (crawlJobChip.querySelector('button[matChipRemove]') as HTMLButtonElement).click();

    expect(removed).toEqual(expect.objectContaining({key: 'crawlJobIdList', value: 'job-1'}));
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
