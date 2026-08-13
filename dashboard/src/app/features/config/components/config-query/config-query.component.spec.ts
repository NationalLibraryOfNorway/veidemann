import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatAutocompleteHarness} from '@angular/material/autocomplete/testing';
import {MatChipListboxHarness} from '@angular/material/chips/testing';
import {MatDialog} from '@angular/material/dialog';
import {of} from 'rxjs';

import {ConfigQuery} from '../../../../shared/func';
import {BrowserScriptType, browserScriptTypes, ConfigObject, Kind, Meta, Role, roles, robotsPolicies, RobotsPolicy} from '../../../../shared/models';
import {LabelService} from '../../services/label.service';
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
  let getLabelKeys: ReturnType<typeof vi.fn>;

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
    getLabelKeys = vi.fn(() => of(['owner', 'category']));
    await TestBed.configureTestingModule({
      imports: [ConfigQueryComponent],
      providers: [
        {provide: MatDialog, useValue: dialog},
        {provide: LabelService, useValue: {getLabelKeys}},
      ],
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

  it('preserves applied label searches when reactive filters change', async () => {
    const emitted: Partial<ConfigQuery>[] = [];
    fixture.componentInstance.queryChange.subscribe(value => emitted.push(value));

    fixture.componentRef.setInput('query', {...query, term: 'label:owner:archive'});
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.componentInstance.form.controls['entityId'].setValue('entity-1');
    expect(emitted.at(-1)?.term).toBe('label:owner:archive');

    fixture.componentRef.setInput('query', {...query, term: 'example label:owner:archive'});
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.componentInstance.form.controls['entityId'].setValue('entity-2');
    expect(emitted.at(-1)?.term).toBe('example label:owner:archive');
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

  it('leaves the state filter to the configuration list header', () => {
    const forms = fixture.nativeElement.querySelectorAll('form') as NodeListOf<HTMLFormElement>;
    const form = forms[0];
    const searchField = fixture.nativeElement.querySelector('.search-form') as HTMLElement;

    expect(forms).toHaveLength(1);
    expect(searchField.parentElement).toBe(form);
    expect(fixture.nativeElement.querySelector('.status-filter')).toBeNull();
    expect(fixture.nativeElement.querySelector('.seed-chip-filters')).toBeNull();
    expect(fixture.nativeElement.querySelector('.crawl-job-filters')).toBeNull();
    expect(fixture.nativeElement.querySelector('[formcontrolname="entityId"]')
      .closest('mat-form-field').hidden).toBe(true);
  });

  it('keeps selected CrawlJobs in the query control without rendering filter chips', async () => {
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

    expect(fixture.componentInstance.form.controls['crawlJobIdList'].value).toEqual(['job-1']);
    expect(fixture.nativeElement.querySelector('.crawl-job-filters')).toBeNull();
    expect(fixture.nativeElement.querySelector('fieldset')).toBeNull();
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
    }
  });

  it('inserts a draft label term and activates label-key suggestions without searching', async () => {
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

    const autocomplete = await loader.getHarness(MatAutocompleteHarness);
    expect(await autocomplete.isOpen()).toBe(true);
    const options = await autocomplete.getOptions();
    expect(await Promise.all(options.map(option => option.getText()))).toEqual(['owner', 'category']);
    expect(getLabelKeys).toHaveBeenCalledWith(Kind.SEED);

    await autocomplete.selectOption({text: 'owner'});
    expect(input.value).toBe('label:owner:');
    expect(emitted).toHaveLength(0);
  });

  it('activates and filters label-key suggestions when label: is typed manually', async () => {
    const autocomplete = await loader.getHarness(MatAutocompleteHarness);

    await autocomplete.enterText('example label:cat');

    expect(await autocomplete.isOpen()).toBe(true);
    const options = await autocomplete.getOptions();
    expect(await Promise.all(options.map(option => option.getText()))).toEqual(['category']);
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
      const searchIcon = fixture.nativeElement.querySelector('.search-form mat-icon[matprefix]') as HTMLElement;
      expect(label.textContent?.trim()).toBe(expectedLabel);
      expect(searchIcon.textContent?.trim()).toBe('search');
      expect(searchIcon.getAttribute('aria-hidden')).toBe('true');
    }

    fixture.componentRef.setInput('query', {...query, kind: Kind.UNDEFINED});
    fixture.detectChanges();
    expect(fixture.componentInstance.searchLabel).toBe('Search');
  });

  it('uses the doubled desktop search width while retaining its shrink limit', () => {
    const searchField = fixture.nativeElement.querySelector('.search-form') as HTMLElement;
    const styles = getComputedStyle(searchField);

    expect(styles.flexBasis).toBe('560px');
    expect(styles.minWidth).toBe('280px');
    expect(styles.maxWidth).toBe('800px');
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
