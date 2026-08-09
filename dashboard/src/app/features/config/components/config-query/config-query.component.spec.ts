import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatChipListboxHarness} from '@angular/material/chips/testing';
import {MatDialog} from '@angular/material/dialog';
import {MatSelectHarness} from '@angular/material/select/testing';
import {of} from 'rxjs';

import {ConfigQuery} from '../../../../shared/func';
import {BrowserScriptType, browserScriptTypes, ConfigObject, Kind, Meta, Role, roles, robotsPolicies, RobotsPolicy} from '../../../../shared/models';
import {ConfigQueryComponent} from './config-query.component';

const searchLabels = [
  [Kind.CRAWLENTITY, 'Search entities'],
  [Kind.SEED, 'Search seeds'],
  [Kind.CRAWLJOB, 'Search crawl jobs'],
  [Kind.CRAWLSCHEDULECONFIG, 'Search schedules'],
  [Kind.CRAWLCONFIG, 'Search crawl configs'],
  [Kind.COLLECTION, 'Search collections'],
  [Kind.BROWSERCONFIG, 'Search browser configs'],
  [Kind.BROWSERSCRIPT, 'Search browser scripts'],
  [Kind.POLITENESSCONFIG, 'Search politeness configs'],
  [Kind.CRAWLHOSTGROUPCONFIG, 'Search crawl host groups'],
  [Kind.ROLEMAPPING, 'Search users'],
] as const;

describe('ConfigQueryComponent', () => {
  let fixture: ComponentFixture<ConfigQueryComponent>;
  let loader: HarnessLoader;
  let dialog: {open: ReturnType<typeof vi.fn>};

  const query: ConfigQuery = {
    kind: Kind.SEED,
    entityId: '',
    scheduleId: '',
    crawlConfigId: '',
    collectionId: '',
    browserConfigId: '',
    politenessId: '',
    disabled: null,
    browserScriptType: null,
    robotsPolicy: null,
    role: null,
    crawlJobIdList: [],
    scriptIdList: [],
    term: '',
    active: '',
    direction: '',
  };

  beforeEach(async () => {
    dialog = {open: vi.fn(() => ({afterClosed: () => of('🐶')}))};
    await TestBed.configureTestingModule({
      imports: [ConfigQueryComponent],
      providers: [{provide: MatDialog, useValue: dialog}],
    }).compileComponents();

    fixture = TestBed.createComponent(ConfigQueryComponent);
    loader = TestbedHarnessEnvironment.loader(fixture);
    fixture.componentRef.setInput('options', {
      browserScripts: [],
      browserConfigs: [],
      browserScriptTypes,
      collections: [],
      crawlConfigs: [],
      crawlJobs: [],
      crawlScheduleConfigs: [],
      politenessConfigs: [],
      robotsPolicies,
      roles,
    });
    fixture.componentRef.setInput('query', query);
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('uses two single-select status chips with no selection for all seeds', async () => {
    const emitted: Partial<ConfigQuery>[] = [];
    fixture.componentInstance.queryChange.subscribe(value => emitted.push(value));
    const listbox = await loader.getHarness(
      MatChipListboxHarness.with({selector: '[formControlName="disabled"]'})
    );
    const [enabled, disabled] = await listbox.getChips();

    expect(await listbox.isMultiple()).toBe(false);
    expect(await Promise.all([enabled.getText(), disabled.getText()])).toEqual(['Show enabled', 'Show disabled']);
    expect(await Promise.all([enabled.isSelected(), disabled.isSelected()])).toEqual([false, false]);
    expect(await loader.getAllHarnesses(
      MatSelectHarness.with({selector: '[formControlName="disabled"]'})
    )).toHaveLength(0);

    await enabled.select();
    expect(emitted.at(-1)?.disabled).toBe(false);
    expect(await Promise.all([enabled.isSelected(), disabled.isSelected()])).toEqual([true, false]);

    await disabled.select();
    expect(emitted.at(-1)?.disabled).toBe(true);
    expect(await Promise.all([enabled.isSelected(), disabled.isSelected()])).toEqual([false, true]);

    await disabled.toggle();
    expect(emitted.at(-1)?.disabled).toBeNull();
    expect(await Promise.all([enabled.isSelected(), disabled.isSelected()])).toEqual([false, false]);
  });

  it('filters role mappings with single-select role chips', async () => {
    const emitted: Partial<ConfigQuery>[] = [];
    fixture.componentInstance.queryChange.subscribe(value => emitted.push(value));
    fixture.componentRef.setInput('query', {...query, kind: Kind.ROLEMAPPING});
    fixture.detectChanges();
    await fixture.whenStable();

    const listbox = await loader.getHarness(
      MatChipListboxHarness.with({selector: '[formControlName="role"]'})
    );
    const roleChips = await listbox.getChips();
    expect(await listbox.isMultiple()).toBe(false);
    expect(await Promise.all(roleChips.map(chip => chip.getText()))).toContain('CURATOR');

    await roleChips[roles.indexOf(Role.CURATOR)].select();
    expect(emitted.at(-1)?.role).toBe(Role.CURATOR);
  });

  it('places Status in the natural form flow without a dedicated Seed chip row', () => {
    const forms = fixture.nativeElement.querySelectorAll('form') as NodeListOf<HTMLFormElement>;
    const form = forms[0];
    const searchField = fixture.nativeElement.querySelector('.search-form') as HTMLElement;
    const statusFilter = fixture.nativeElement.querySelector('.status-filter') as HTMLFieldSetElement;

    expect(forms).toHaveLength(1);
    expect(searchField.parentElement).toBe(form);
    expect(statusFilter.parentElement).toBe(form);
    expect(statusFilter.querySelectorAll('mat-chip-option')).toHaveLength(2);
    expect(fixture.nativeElement.querySelector('.seed-chip-filters')).toBeNull();
    expect(fixture.nativeElement.querySelector('.crawl-job-filters')).toBeNull();
  });

  it('shows one selected CrawlJob after Status without a full-width wrapper', async () => {
    fixture.componentRef.setInput('options', {
      ...fixture.componentInstance.options,
      crawlJobs: [
        new ConfigObject({id: 'job-1', kind: Kind.CRAWLJOB, meta: new Meta({name: 'Daily job'})}),
      ],
    });
    fixture.componentRef.setInput('query', {
      ...query,
      crawlJobIdList: ['job-1'],
    });
    fixture.detectChanges();
    await fixture.whenStable();

    const form = fixture.nativeElement.querySelector('form') as HTMLFormElement;
    const crawlJobFilters = form.querySelector('.crawl-job-filters') as HTMLFieldSetElement;
    const statusFilter = form.querySelector('.status-filter') as HTMLFieldSetElement;
    const text = crawlJobFilters.textContent?.replace(/\s+/g, ' ');

    expect(crawlJobFilters.querySelector('legend')?.textContent?.trim()).toBe('Crawl jobs');
    expect(crawlJobFilters.querySelectorAll('mat-chip')).toHaveLength(1);
    expect(text).toContain('Daily job');
    expect(text).not.toContain('Crawljob:');
    expect(statusFilter.parentElement).toBe(form);
    expect(crawlJobFilters.parentElement).toBe(form);
    expect(statusFilter.nextElementSibling).toBe(crawlJobFilters);
    expect(fixture.nativeElement.querySelector('.seed-chip-filters')).toBeNull();
  });

  it('removes selected CrawlJobs through the reactive query flow', async () => {
    const emitted: Partial<ConfigQuery>[] = [];
    fixture.componentInstance.queryChange.subscribe(value => emitted.push(value));
    fixture.componentRef.setInput('options', {
      ...fixture.componentInstance.options,
      crawlJobs: [
        new ConfigObject({id: 'job-1', kind: Kind.CRAWLJOB, meta: new Meta({name: 'Daily job'})}),
      ],
    });
    fixture.componentRef.setInput('query', {
      ...query,
      entityId: 'entity-1',
      disabled: false,
      crawlJobIdList: ['job-1', 'job-missing'],
    });
    fixture.detectChanges();
    await fixture.whenStable();

    const removeButton = fixture.nativeElement.querySelector(
      '.crawl-job-filters button[matChipRemove]'
    ) as HTMLButtonElement;
    expect(removeButton.getAttribute('aria-label')).toBe('Remove Daily job crawl job filter');
    removeButton.click();
    fixture.detectChanges();
    await fixture.whenStable();

    expect(emitted.at(-1)).toEqual(expect.objectContaining({
      entityId: 'entity-1',
      disabled: false,
      crawlJobIdList: ['job-missing'],
    }));
  });

  it('refreshes selected CrawlJob chips from external queries', async () => {
    fixture.componentRef.setInput('query', {...query, crawlJobIdList: ['job-1', 'job-2']});
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelectorAll('.crawl-job-filters mat-chip')).toHaveLength(2);

    fixture.componentRef.setInput('query', {...query, crawlJobIdList: ['job-2']});
    fixture.detectChanges();
    await fixture.whenStable();

    const chips = fixture.nativeElement.querySelectorAll('.crawl-job-filters mat-chip') as NodeListOf<HTMLElement>;
    expect(chips).toHaveLength(1);
    expect(chips[0].textContent).toContain('job-2');
  });

  it('searches only on Enter and clears the search immediately', async () => {
    const emitted: Partial<ConfigQuery>[] = [];
    fixture.componentInstance.queryChange.subscribe(value => emitted.push(value));
    const input = fixture.nativeElement.querySelector('.search-form input') as HTMLInputElement;

    input.value = 'example.org';
    input.dispatchEvent(new Event('input', {bubbles: true}));
    fixture.detectChanges();
    await fixture.whenStable();
    expect(emitted).toHaveLength(0);

    input.dispatchEvent(new KeyboardEvent('keyup', {key: 'Enter', bubbles: true}));
    fixture.detectChanges();
    await fixture.whenStable();
    expect(emitted.at(-1)?.term).toBe('example.org');

    const clear = fixture.nativeElement.querySelector('[data-testid="clear-search"]') as HTMLButtonElement;
    clear.click();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(emitted.at(-1)?.term).toBe('');
  });

  it('offers compact label and emoji search helpers for configuration lists', async () => {
    const supportedKinds = searchLabels
      .map(([kind]) => kind)
      .filter(kind => kind !== Kind.ROLEMAPPING);

    for (const kind of supportedKinds) {
      fixture.componentRef.setInput('query', {...query, kind});
      fixture.detectChanges();
      await fixture.whenStable();

      expect(fixture.nativeElement.querySelector('[data-testid="label-search-helper"]')).not.toBeNull();
      expect(fixture.nativeElement.querySelector('[data-testid="emoji-search-helper"]')).not.toBeNull();
      const helperGroup = fixture.nativeElement.querySelector('.search-helper-actions') as HTMLElement;
      expect(helperGroup.querySelectorAll('button')).toHaveLength(2);
      expect(fixture.nativeElement.querySelector('.label-search-suggestions')).toBeNull();
    }
  });

  it('inserts a draft label term without searching and hides the helpers', async () => {
    const emitted: Partial<ConfigQuery>[] = [];
    fixture.componentInstance.queryChange.subscribe(value => emitted.push(value));
    const input = fixture.nativeElement.querySelector('.search-form input') as HTMLInputElement;
    const labelHelper = fixture.nativeElement.querySelector(
      '[data-testid="label-search-helper"]'
    ) as HTMLButtonElement;
    labelHelper.click();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(input.value).toBe('label:');
    expect(document.activeElement).toBe(input);
    expect(emitted).toHaveLength(0);
    expect(fixture.nativeElement.querySelector('[data-testid="label-search-helper"]')).toBeNull();
    expect(fixture.nativeElement.querySelector('[data-testid="emoji-search-helper"]')).toBeNull();
    expect(fixture.nativeElement.querySelector('[data-testid="clear-search"]')).not.toBeNull();
  });

  it('searches immediately after choosing an emoji and leaves applied rendering outside the field', async () => {
    const emitted: Partial<ConfigQuery>[] = [];
    fixture.componentInstance.queryChange.subscribe(value => emitted.push(value));
    const input = fixture.nativeElement.querySelector('.search-form input') as HTMLInputElement;

    await fixture.componentInstance.chooseLabelEmoji();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(dialog.open).toHaveBeenCalledOnce();
    expect(dialog.open.mock.calls[0][1]).toEqual(expect.objectContaining({
      width: '552px',
      maxWidth: 'calc(100vw - 24px)',
    }));
    expect(input.value).toBe('label:emoji:🐶');
    expect(document.activeElement).toBe(input);
    expect(emitted).toHaveLength(1);
    expect(emitted[0]).toEqual(expect.objectContaining({
      term: 'label:emoji:🐶',
    }));

    fixture.componentRef.setInput('query', {...query, term: 'label:emoji:🐶'});
    fixture.detectChanges();
    await fixture.whenStable();

    expect(input.value).toBe('');
    expect(fixture.nativeElement.querySelector('mat-chip')).toBeNull();
    expect(fixture.nativeElement.querySelector('[data-testid="clear-search"]')).toBeNull();
    expect(fixture.nativeElement.querySelector('[data-testid="label-search-helper"]')).toBeNull();
    expect(fixture.nativeElement.querySelector('[data-testid="emoji-search-helper"]')).toBeNull();
  });

  it('shows only the editable name portion of applied label queries', async () => {
    fixture.componentRef.setInput('query', {...query, term: 'label:owner:archive'});
    fixture.detectChanges();
    await fixture.whenStable();
    let input = fixture.nativeElement.querySelector('.search-form input') as HTMLInputElement;
    expect(input.value).toBe('');

    fixture.componentRef.setInput('query', {...query, term: 'label:owner'});
    fixture.detectChanges();
    await fixture.whenStable();
    expect(input.value).toBe('');

    fixture.componentRef.setInput('query', {...query, term: 'example label:owner:archive:2026'});
    fixture.detectChanges();
    await fixture.whenStable();
    input = fixture.nativeElement.querySelector('.search-form input') as HTMLInputElement;
    expect(input.value).toBe('example');
    expect(fixture.nativeElement.querySelector('[data-testid="clear-search"]')).not.toBeNull();
  });

  it('keeps an incomplete label expression as editable text', async () => {
    fixture.componentRef.setInput('query', {...query, term: 'label:'});
    fixture.detectChanges();
    await fixture.whenStable();

    const input = fixture.nativeElement.querySelector('.search-form input') as HTMLInputElement;
    expect(input.value).toBe('label:');
    expect(fixture.nativeElement.querySelector('.search-form mat-chip')).toBeNull();
    expect(fixture.nativeElement.querySelector('[data-testid="clear-search"]')).not.toBeNull();
  });

  it('reconstructs combined queries and Clear preserves the applied label', async () => {
    const emitted: Partial<ConfigQuery>[] = [];
    fixture.componentInstance.queryChange.subscribe(value => emitted.push(value));
    fixture.componentRef.setInput('query', {...query, term: 'example label:owner:archive'});
    fixture.detectChanges();
    await fixture.whenStable();

    const input = fixture.nativeElement.querySelector('.search-form input') as HTMLInputElement;
    input.dispatchEvent(new KeyboardEvent('keyup', {key: 'Enter', bubbles: true}));
    expect(emitted.at(-1)?.term).toBe('example label:owner:archive');

    const clear = fixture.nativeElement.querySelector('[data-testid="clear-search"]') as HTMLButtonElement;
    clear.click();
    expect(emitted.at(-1)?.term).toBe('label:owner:archive');
    expect(fixture.componentInstance.appliedLabelSearch?.selector).toBe('owner:archive');
    expect(document.activeElement).toBe(input);
  });

  it('keeps Users email and group search without label helpers', async () => {
    const emitted: Partial<ConfigQuery>[] = [];
    fixture.componentInstance.queryChange.subscribe(value => emitted.push(value));
    fixture.componentRef.setInput('query', {...query, kind: Kind.ROLEMAPPING});
    fixture.detectChanges();
    await fixture.whenStable();

    expect(fixture.nativeElement.querySelector('[data-testid="label-search-helper"]')).toBeNull();
    expect(fixture.nativeElement.querySelector('[data-testid="emoji-search-helper"]')).toBeNull();

    const input = fixture.nativeElement.querySelector('.search-form input') as HTMLInputElement;
    input.value = 'group:administrators';
    input.dispatchEvent(new Event('input', {bubbles: true}));
    input.dispatchEvent(new KeyboardEvent('keyup', {key: 'Enter', bubbles: true}));
    fixture.detectChanges();
    await fixture.whenStable();

    expect(emitted.at(-1)?.term).toBe('group:administrators');
  });

  it('does not show label helpers for an undefined configuration route', () => {
    fixture.componentRef.setInput('query', {...query, kind: Kind.UNDEFINED});
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('[data-testid="label-search-helper"]')).toBeNull();
    expect(fixture.nativeElement.querySelector('[data-testid="emoji-search-helper"]')).toBeNull();
  });

  it('describes the active configuration type in the search label', async () => {
    for (const [kind, expectedLabel] of searchLabels) {
      fixture.componentRef.setInput('query', {...query, kind});
      fixture.detectChanges();
      await fixture.whenStable();

      const label = fixture.nativeElement.querySelector('.search-form mat-label') as HTMLElement;
      expect(label.textContent?.trim()).toBe(expectedLabel);
    }

    fixture.componentRef.setInput('query', {...query, kind: Kind.UNDEFINED});
    fixture.detectChanges();
    expect(fixture.componentInstance.searchLabel).toBe('Search');
  });

  it('updates the selected status chip from external queries', async () => {
    const listbox = await loader.getHarness(MatChipListboxHarness);
    const [enabled, disabled] = await listbox.getChips();

    fixture.componentRef.setInput('query', {...query, disabled: false});
    fixture.detectChanges();
    await fixture.whenStable();
    expect(await Promise.all([enabled.isSelected(), disabled.isSelected()])).toEqual([true, false]);

    fixture.componentRef.setInput('query', {...query, disabled: true});
    fixture.detectChanges();
    await fixture.whenStable();
    expect(await Promise.all([enabled.isSelected(), disabled.isSelected()])).toEqual([false, true]);

    fixture.componentRef.setInput('query', {...query, disabled: null});
    fixture.detectChanges();
    await fixture.whenStable();
    expect(await Promise.all([enabled.isSelected(), disabled.isSelected()])).toEqual([false, false]);
  });

  it('uses the same deselectable status chips for CrawlJob as Seed', async () => {
    const emitted: Partial<ConfigQuery>[] = [];
    fixture.componentInstance.queryChange.subscribe(value => emitted.push(value));
    fixture.componentRef.setInput('query', {...query, kind: Kind.CRAWLJOB});
    fixture.detectChanges();
    await fixture.whenStable();

    const listbox = await loader.getHarness(MatChipListboxHarness.with({selector: '[formControlName="disabled"]'}));
    const chips = await listbox.getChips();
    expect(await Promise.all(chips.map(chip => chip.getText()))).toEqual(['Show enabled', 'Show disabled']);
    expect(await Promise.all(chips.map(chip => chip.isSelected()))).toEqual([false, false]);
    await chips[0].select();
    expect(fixture.componentInstance.form.controls['disabled'].value).toBe(false);
    await chips[0].toggle();
    expect(emitted.at(-1)?.disabled).toBeNull();
  });

  it('filters BrowserScripts with four single-select type chips and supports clearing the selection', async () => {
    const emitted: Partial<ConfigQuery>[] = [];
    fixture.componentInstance.queryChange.subscribe(value => emitted.push(value));
    fixture.componentRef.setInput('query', {...query, kind: Kind.BROWSERSCRIPT});
    fixture.detectChanges();
    await fixture.whenStable();

    const listbox = await loader.getHarness(
      MatChipListboxHarness.with({selector: '[formControlName="browserScriptType"]'})
    );
    const chips = await listbox.getChips();

    expect(await listbox.isMultiple()).toBe(false);
    expect(await Promise.all(chips.map(chip => chip.getText()))).toEqual([
      'EXTRACT_OUTLINKS',
      'ON_LOAD',
      'ON_NEW_DOCUMENT',
      'SCOPE_CHECK',
    ]);
    expect(await Promise.all(chips.map(chip => chip.isSelected()))).toEqual([false, false, false, false]);

    await chips[1].select();
    expect(emitted.at(-1)?.browserScriptType).toBe(BrowserScriptType.ON_LOAD);
    expect(await chips[1].isSelected()).toBe(true);

    await chips[1].toggle();
    expect(emitted.at(-1)?.browserScriptType).toBeNull();
    expect(await chips[1].isSelected()).toBe(false);

    fixture.componentRef.setInput('query', {
      ...query,
      kind: Kind.BROWSERSCRIPT,
      browserScriptType: BrowserScriptType.SCOPE_CHECK,
    });
    fixture.detectChanges();
    await fixture.whenStable();
    expect(await chips[3].isSelected()).toBe(true);
  });

  it('filters politeness configurations with a deselectable robot-policy chip list', async () => {
    const emitted: Partial<ConfigQuery>[] = [];
    fixture.componentInstance.queryChange.subscribe(value => emitted.push(value));
    fixture.componentRef.setInput('query', {...query, kind: Kind.POLITENESSCONFIG});
    fixture.detectChanges();
    await fixture.whenStable();

    const listbox = await loader.getHarness(MatChipListboxHarness.with({selector: '[formControlName="robotsPolicy"]'}));
    const chips = await listbox.getChips();
    expect(chips).toHaveLength(robotsPolicies.length);
    await chips[RobotsPolicy.IGNORE_ROBOTS].select();
    expect(emitted.at(-1)?.robotsPolicy).toBe(RobotsPolicy.IGNORE_ROBOTS);
    await chips[RobotsPolicy.IGNORE_ROBOTS].toggle();
    expect(emitted.at(-1)?.robotsPolicy).toBeNull();
  });
});
