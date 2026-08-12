import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';

import {provideMaterialAnimationsDisabled} from '../../../../core/core.testing.module';
import {ConfigObject, ConfigRef, CrawlConfig, CrawlJob, Kind, Meta, Seed} from '../../../../shared/models';
import {ConfigShortcutHelpersComponent} from './shortcut.component';
import {of} from 'rxjs';
import {ConfigService} from '../../../../shared/services';

describe('ConfigShortcutHelpersComponent', () => {
  let fixture: ComponentFixture<ConfigShortcutHelpersComponent>;
  let canReadSeeds: boolean;
  let canCreateEntity: boolean;
  let canCreateSeed: boolean;
  let canDeleteEntity: boolean;
  let canUpdateEntity: boolean;
  let canRunSeed: boolean;
  let canRunCrawlJob: boolean;
  let configNames: Record<string, string>;
  let configObjects: Record<string, ConfigObject>;
  const getConfig = vi.fn((ref: ConfigRef) => of(configObjects[ref.id] || new ConfigObject({
    id: ref.id,
    kind: ref.kind,
    meta: new Meta({name: configNames[ref.id] || ref.id}),
  })));

  beforeEach(async () => {
    canReadSeeds = true;
    canCreateEntity = false;
    canCreateSeed = false;
    canDeleteEntity = false;
    canUpdateEntity = false;
    canRunSeed = false;
    canRunCrawlJob = false;
    configNames = {};
    configObjects = {};
    getConfig.mockClear();

    await TestBed.configureTestingModule({
      imports: [ConfigShortcutHelpersComponent],
      providers: [
        provideMaterialAnimationsDisabled(),
        provideRouter([]),
        {
          provide: ConfigService,
          useValue: {get: getConfig},
        },
        {
          provide: AbilityServiceSignal,
          useValue: {
            can: (action: string, subject: string) =>
              (action === 'read' && subject === Kind[Kind.SEED] && canReadSeeds)
              || (action === 'read' && [
                Kind[Kind.COLLECTION],
                Kind[Kind.BROWSERCONFIG],
                Kind[Kind.POLITENESSCONFIG],
                Kind[Kind.CRAWLJOB],
                Kind[Kind.CRAWLCONFIG],
                Kind[Kind.CRAWLSCHEDULECONFIG],
                Kind[Kind.CRAWLENTITY],
              ].includes(subject))
              || (action === 'create' && subject === Kind[Kind.CRAWLENTITY] && canCreateEntity)
              || (action === 'create' && subject === Kind[Kind.SEED] && canCreateSeed)
              || (action === 'update' && subject === Kind[Kind.CRAWLENTITY] && canUpdateEntity)
              || (action === 'runCrawl' && subject === Kind[Kind.SEED] && canRunSeed)
              || (action === 'runCrawl' && subject === Kind[Kind.CRAWLJOB] && canRunCrawlJob)
              || (action === 'delete' && subject === Kind[Kind.CRAWLENTITY] && canDeleteEntity),
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'entity-1',
      kind: Kind.CRAWLENTITY,
    }));
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('links from an entity detail to its filtered seed list', async () => {
    (fixture.nativeElement.querySelector('[data-testid="config-actions-menu"]') as HTMLButtonElement).click();
    fixture.detectChanges();
    await fixture.whenStable();
    const links = Array.from<HTMLAnchorElement>(document.querySelectorAll('a[mat-menu-item]'));
    const showSeedsLink = links.find(link => link.textContent.includes('Show seeds'));

    expect(showSeedsLink).toBeDefined();
    expect(showSeedsLink.getAttribute('href')).toContain('seed?entity_id=entity-1');
    expect(showSeedsLink.classList.contains('mat-mdc-chip')).toBe(false);
    expect(showSeedsLink.hasAttribute('mat-menu-item')).toBe(true);
    expect(showSeedsLink.querySelector('mat-icon').textContent.trim()).toBe('link');
    expect(fixture.nativeElement.querySelector('.shortcut-group, .group-label')).toBeNull();
    expect(fixture.nativeElement.querySelector('.more-actions')).toBeNull();
  });

  it('does not duplicate the configuration ID as a shortcut chip', () => {
    expect(fixture.nativeElement.querySelector('.main-id-chip')).toBeNull();
    expect(fixture.nativeElement.textContent).not.toContain('entity-1');
  });

  it('hides the entity seed shortcut without seed read permission', () => {
    canReadSeeds = false;
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'entity-1',
      kind: Kind.CRAWLENTITY,
    }));
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).not.toContain('Show seeds');
    expect(fixture.nativeElement.querySelector('.more-actions')).toBeNull();
  });

  it('puts Edit, Clone, and Delete in the configuration actions menu', async () => {
    canCreateEntity = true;
    canCreateSeed = true;
    canDeleteEntity = true;
    canUpdateEntity = true;
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    const entity = new ConfigObject({id: 'entity-1', kind: Kind.CRAWLENTITY});
    fixture.componentRef.setInput('configObject', entity);
    fixture.componentRef.setInput('showEdit', true);
    fixture.detectChanges();
    const cloned: ConfigObject[] = [];
    const deleted: ConfigObject[] = [];
    const edited: ConfigObject[] = [];
    const createdFrom: ConfigObject[] = [];
    fixture.componentInstance.clone.subscribe(value => cloned.push(value));
    fixture.componentInstance.delete.subscribe(value => deleted.push(value));
    fixture.componentInstance.edit.subscribe(value => edited.push(value));
    fixture.componentInstance.createSeed.subscribe(value => createdFrom.push(value));

    const menuTrigger = fixture.nativeElement.querySelector(
      '[data-testid="config-actions-menu"]'
    ) as HTMLButtonElement;
    expect(menuTrigger).not.toBeNull();
    expect(fixture.nativeElement.querySelector('mat-chip[role="button"]')).toBeNull();
    menuTrigger.click();
    fixture.detectChanges();
    await fixture.whenStable();
    const createSeedMenuItem = document.querySelector('[data-testid="create-seed-action"]') as HTMLButtonElement;
    createSeedMenuItem.click();
    fixture.detectChanges();
    await fixture.whenStable();
    menuTrigger.click();
    fixture.detectChanges();
    await fixture.whenStable();
    const editMenuItem = document.querySelector('[data-testid="edit-config-action"]') as HTMLButtonElement;
    editMenuItem.click();
    fixture.detectChanges();
    await fixture.whenStable();
    menuTrigger.click();
    fixture.detectChanges();
    await fixture.whenStable();
    const cloneMenuItem = document.querySelector('[data-testid="clone-config-action"]') as HTMLButtonElement;
    cloneMenuItem.click();
    fixture.detectChanges();
    await fixture.whenStable();
    menuTrigger.click();
    fixture.detectChanges();
    await fixture.whenStable();
    const deleteMenuItem = document.querySelector('[data-testid="delete-config-action"]') as HTMLButtonElement;
    deleteMenuItem.click();

    expect(fixture.nativeElement.querySelector('.shortcut-group, .group-label')).toBeNull();
    expect(cloned).toEqual([entity]);
    expect(deleted).toEqual([entity]);
    expect(edited).toEqual([entity]);
    expect(createdFrom).toEqual([entity]);
  });

  it('can hide Clone and Delete for related cards without hiding other context actions', async () => {
    canCreateEntity = true;
    canCreateSeed = true;
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    fixture.componentRef.setInput('configObject', new ConfigObject({id: 'entity-1', kind: Kind.CRAWLENTITY}));
    fixture.componentRef.setInput('showClone', false);
    fixture.componentRef.setInput('showDelete', false);
    fixture.detectChanges();

    const menuTrigger = fixture.nativeElement.querySelector(
      '[data-testid="config-actions-menu"]'
    ) as HTMLButtonElement;
    expect(menuTrigger).not.toBeNull();
    expect(fixture.nativeElement.textContent).not.toContain('Create Seed');
    menuTrigger.click();
    fixture.detectChanges();
    await fixture.whenStable();

    expect(document.querySelector('[data-testid="clone-config-action"]')).toBeNull();
    expect(document.querySelector('[data-testid="delete-config-action"]')).toBeNull();
    expect(document.querySelector('[data-testid="create-seed-action"]')).not.toBeNull();
  });

  it('loads referenced configurations when related option collections are unavailable', async () => {
    configNames = {
      'collection-1': 'News collection',
      'browser-config-1': 'Desktop browser',
      'politeness-1': 'Public sites policy',
    };
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'crawl-config-1',
      kind: Kind.CRAWLCONFIG,
      crawlConfig: new CrawlConfig({
        collectionRef: new ConfigRef({kind: Kind.COLLECTION, id: 'collection-1'}),
        browserConfigRef: new ConfigRef({kind: Kind.BROWSERCONFIG, id: 'browser-config-1'}),
        politenessRef: new ConfigRef({kind: Kind.POLITENESSCONFIG, id: 'politeness-1'}),
      }),
    }));
    fixture.detectChanges();
    await fixture.whenStable();

    expect(fixture.nativeElement.textContent).toContain('News collection');
    expect(fixture.nativeElement.textContent).toContain('Desktop browser');
    expect(fixture.nativeElement.textContent).toContain('Public sites policy');
    expect(getConfig.mock.calls.map(([ref]) => `${ref.kind}:${ref.id}`)).toEqual(expect.arrayContaining([
      `${Kind.COLLECTION}:collection-1`,
      `${Kind.BROWSERCONFIG}:browser-config-1`,
      `${Kind.POLITENESSCONFIG}:politeness-1`,
    ]));
  });

  it('uses only the relationship names as standard links in the accessible relationship navigation', async () => {
    configNames = {
      'collection-1': 'News collection',
      'browser-config-1': 'Desktop browser',
      'politeness-1': 'Public sites policy',
    };
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'crawl-config-1',
      kind: Kind.CRAWLCONFIG,
      crawlConfig: new CrawlConfig({
        collectionRef: new ConfigRef({kind: Kind.COLLECTION, id: 'collection-1'}),
        browserConfigRef: new ConfigRef({kind: Kind.BROWSERCONFIG, id: 'browser-config-1'}),
        politenessRef: new ConfigRef({kind: Kind.POLITENESSCONFIG, id: 'politeness-1'}),
      }),
    }));
    fixture.detectChanges();
    await fixture.whenStable();

    const text = fixture.nativeElement.textContent;
    expect(text).toContain('Collection');
    expect(text).toContain('Browser config');
    expect(text).toContain('Politeness config');
    expect(text).toContain('News collection');
    expect(text).toContain('Desktop browser');
    expect(text).toContain('Public sites policy');
    const rows = [...fixture.nativeElement.querySelectorAll('.relationship-row')] as HTMLElement[];
    const links = [...fixture.nativeElement.querySelectorAll('.relationship-link')] as HTMLAnchorElement[];
    const navigation = fixture.nativeElement.querySelector('nav.reference-shortcuts') as HTMLElement;
    expect(navigation.getAttribute('aria-label')).toBe('Related configurations');
    expect(links.map(link => link.textContent.trim())).toEqual([
      'News collection',
      'Desktop browser',
      'Public sites policy',
    ]);
    expect(links.map(link => link.getAttribute('href'))).toEqual([
      '/config/collection/collection-1',
      '/config/browserconfig/browser-config-1',
      '/config/politenessconfig/politeness-1',
    ]);
    expect(links.every(link => !link.querySelector('mat-icon, small'))).toBeTruthy();
    expect(rows.every(row => !!row.querySelector('.relationship-content > mat-icon'))).toBeTruthy();
    expect(fixture.nativeElement.querySelector('.navigate-icon')).toBeNull();
    expect(fixture.nativeElement.querySelector('mat-chip-set')).toBeNull();
  });

  it('keeps the deactivated badge in the trailing column without navigation carets', async () => {
    const activeJob = new ConfigObject({
      id: 'active-job',
      kind: Kind.CRAWLJOB,
      meta: new Meta({name: 'Active'}),
      crawlJob: new CrawlJob({disabled: false}),
    });
    const inactiveJob = new ConfigObject({
      id: 'inactive-job',
      kind: Kind.CRAWLJOB,
      meta: new Meta({name: 'Inactive'}),
      crawlJob: new CrawlJob({disabled: true}),
    });
    configObjects = {
      [activeJob.id]: activeJob,
      [inactiveJob.id]: inactiveJob,
    };
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'seed-1',
      kind: Kind.SEED,
      seed: new Seed({jobRefList: [
        ConfigObject.toConfigRef(activeJob),
        ConfigObject.toConfigRef(inactiveJob),
      ]}),
    }));
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const links = [...fixture.nativeElement.querySelectorAll('.relationship-link')] as HTMLElement[];
    const rows = [...fixture.nativeElement.querySelectorAll('.relationship-row')] as HTMLElement[];
    const status = fixture.nativeElement.querySelector('.relation-status') as HTMLElement;
    expect(links).toHaveLength(2);
    expect(rows).toHaveLength(2);
    expect(status.textContent.trim()).toBe('Deactivated');
    expect(fixture.nativeElement.querySelector('.navigate-icon')).toBeNull();
    expect(getComputedStyle(status).gridColumnStart).toBe('3');
    expect(getComputedStyle(status).color).toBe(getComputedStyle(rows[1].querySelector('small')).color);
    expect(getComputedStyle(status).backgroundColor)
      .not.toBe(getComputedStyle(document.documentElement).getPropertyValue('--mat-sys-error-container').trim());
    expect(getComputedStyle(rows[0].querySelector('.relationship-content')).borderRadius).toBe('0px');
    expect(getComputedStyle(rows[0]).borderTopStyle).toBe('none');
    expect(getComputedStyle(rows[0]).borderBottomStyle).toBe('none');
    expect(getComputedStyle(rows[1]).borderTopStyle).toBe('solid');
    expect(getComputedStyle(rows[1]).borderBottomStyle).toBe('none');
  });

  it('opens the full target configuration menu from a relationship row', async () => {
    canCreateEntity = true;
    canCreateSeed = true;
    canDeleteEntity = true;
    canUpdateEntity = true;
    const entity = new ConfigObject({
      id: 'entity-1',
      kind: Kind.CRAWLENTITY,
      meta: new Meta({name: 'Example entity'}),
    });
    configObjects = {[entity.id]: entity};
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'seed-1',
      kind: Kind.SEED,
      seed: new Seed({entityRef: ConfigObject.toConfigRef(entity)}),
    }));
    fixture.componentRef.setInput('showFilters', false);
    fixture.componentRef.setInput('showActions', false);
    fixture.componentRef.setInput('showCreateSeed', false);
    const edited: ConfigObject[] = [];
    fixture.componentInstance.edit.subscribe(value => edited.push(value));
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const row = fixture.nativeElement.querySelector('.relationship-row') as HTMLElement;
    const trigger = row.querySelector('.relationship-menu-trigger') as HTMLButtonElement;
    expect(trigger).not.toBeNull();
    expect(row.lastElementChild).toBe(trigger);
    trigger.click();
    fixture.detectChanges();
    await fixture.whenStable();

    const menu = document.querySelector('.mat-mdc-menu-panel') as HTMLElement;
    expect(menu.textContent).toContain('Show seeds');
    expect(menu.querySelector('[data-testid="create-seed-action"]')).not.toBeNull();
    expect(menu.querySelector('[data-testid="edit-config-action"]')).not.toBeNull();
    expect(menu.querySelector('[data-testid="clone-config-action"]')).not.toBeNull();
    expect(menu.querySelector('[data-testid="delete-config-action"]')).not.toBeNull();
    (menu.querySelector('[data-testid="edit-config-action"]') as HTMLButtonElement).click();
    expect(edited).toEqual([entity]);
  });

  it('puts the direct seed crawl action in the menu', async () => {
    canRunSeed = true;
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    const seed = new ConfigObject({id: 'seed-1', kind: Kind.SEED});
    fixture.componentRef.setInput('configObject', seed);
    const emitted: ConfigObject[] = [];
    fixture.componentInstance.runCrawl.subscribe(value => emitted.push(value));
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('mat-chip[role="button"]')).toBeNull();
    (fixture.nativeElement.querySelector('[data-testid="config-actions-menu"]') as HTMLButtonElement).click();
    fixture.detectChanges();
    await fixture.whenStable();
    const runMenuItem = document.querySelector('[data-testid="run-crawl-action"]') as HTMLButtonElement;
    expect(runMenuItem.textContent).toContain('Run crawl');
    runMenuItem.click();

    expect(emitted).toEqual([seed]);
  });

  it('uses consistent filtered-list wording and target icons in the seed menu', async () => {
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'seed-1',
      kind: Kind.SEED,
      seed: new Seed({
        entityRef: new ConfigRef({kind: Kind.CRAWLENTITY, id: 'entity-1'}),
        jobRefList: [new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-1'})],
      }),
    }));
    fixture.componentRef.setInput('showReferences', false);
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('[data-testid="config-actions-menu"]') as HTMLButtonElement).click();
    fixture.detectChanges();
    await fixture.whenStable();

    const items = [...document.querySelectorAll<HTMLAnchorElement>('.mat-mdc-menu-panel a[mat-menu-item]')];
    const seedListItems = items.filter(item => item.textContent.includes('Show seeds with the same'));
    expect(seedListItems.map(item => item.querySelector('span').textContent.trim())).toEqual([
      'Show seeds with the same crawl job',
      'Show seeds with the same entity',
    ]);
    expect(seedListItems.map(item => item.querySelector('mat-icon').textContent.trim())).toEqual(['link', 'link']);
  });

  it('uses Go to wording and target icons for detail navigation', async () => {
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'job-1',
      kind: Kind.CRAWLJOB,
      crawlJob: new CrawlJob({
        scheduleRef: new ConfigRef({kind: Kind.CRAWLSCHEDULECONFIG, id: 'schedule-1'}),
        crawlConfigRef: new ConfigRef({kind: Kind.CRAWLCONFIG, id: 'crawl-config-1'}),
      }),
    }));
    fixture.componentRef.setInput('showReferences', false);
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('[data-testid="config-actions-menu"]') as HTMLButtonElement).click();
    fixture.detectChanges();
    await fixture.whenStable();

    const items = [...document.querySelectorAll<HTMLAnchorElement>('.mat-mdc-menu-panel a[mat-menu-item]')];
    const detailItems = items.filter(item => item.textContent.includes('Go to'));
    expect(detailItems.map(item => item.querySelector('span').textContent.trim())).toEqual([
      'Go to schedule',
      'Go to crawl configuration',
    ]);
    expect(detailItems.map(item => item.querySelector('mat-icon').textContent.trim())).toEqual([
      'schedule',
      'settings_system_daydream',
    ]);
  });

  it('puts the direct crawljob run action in the menu', async () => {
    canRunCrawlJob = true;
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    const crawlJob = new ConfigObject({
      id: 'job-1',
      kind: Kind.CRAWLJOB,
      crawlJob: new CrawlJob(),
    });
    fixture.componentRef.setInput('configObject', crawlJob);
    const emitted: ConfigObject[] = [];
    fixture.componentInstance.runCrawl.subscribe(value => emitted.push(value));
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('[data-testid="config-actions-menu"]') as HTMLButtonElement).click();
    fixture.detectChanges();
    await fixture.whenStable();
    const runMenuItem = document.querySelector('[data-testid="run-crawl-action"]') as HTMLButtonElement;
    expect(runMenuItem.textContent).toContain('Run crawl');
    runMenuItem.click();

    expect(emitted).toEqual([crawlJob]);
  });

  it('puts the related crawljob seed action in the menu and emits both contexts', async () => {
    canRunSeed = true;
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    const seed = new ConfigObject({id: 'seed-1', kind: Kind.SEED});
    const crawlJob = new ConfigObject({
      id: 'job-1',
      kind: Kind.CRAWLJOB,
      meta: new Meta({name: 'Daily crawl'}),
      crawlJob: new CrawlJob(),
    });
    fixture.componentRef.setInput('configObject', crawlJob);
    fixture.componentRef.setInput('seedContext', seed);
    const emitted: {seed: ConfigObject; crawlJob: ConfigObject}[] = [];
    fixture.componentInstance.runSeedInCrawlJob.subscribe(value => emitted.push(value));
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.action-shortcuts')).toBeNull();
    const menuTrigger = fixture.nativeElement.querySelector(
      '[data-testid="config-actions-menu"]'
    ) as HTMLButtonElement;
    menuTrigger.click();
    fixture.detectChanges();
    await fixture.whenStable();
    const runMenuItem = document.querySelector('[data-testid="run-crawl-action"]') as HTMLButtonElement;
    expect(runMenuItem.textContent).toContain('Run crawl with this crawl job');
    runMenuItem.click();

    expect(emitted).toEqual([{seed, crawlJob}]);
  });

  it('does not authorize a related seed crawl with crawljob permission alone', async () => {
    canRunCrawlJob = true;
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'job-1',
      kind: Kind.CRAWLJOB,
      crawlJob: new CrawlJob(),
    }));
    fixture.componentRef.setInput('seedContext', new ConfigObject({id: 'seed-1', kind: Kind.SEED}));
    fixture.componentRef.setInput('showClone', false);
    fixture.componentRef.setInput('showDelete', false);
    fixture.detectChanges();

    const trigger = fixture.nativeElement.querySelector(
      '[data-testid="config-actions-menu"]'
    ) as HTMLButtonElement;
    expect(trigger).not.toBeNull();
    trigger.click();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(document.querySelector('[data-testid="run-crawl-action"]')).toBeNull();
  });
});
