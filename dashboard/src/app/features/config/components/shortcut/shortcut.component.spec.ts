import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';

import {provideMaterialAnimationsDisabled} from '../../../../core/core.testing.module';
import {ConfigObject, ConfigRef, CrawlConfig, CrawlJob, Kind, Meta} from '../../../../shared/models';
import {ConfigShortcutHelpersComponent} from './shortcut.component';

describe('ConfigShortcutHelpersComponent', () => {
  let fixture: ComponentFixture<ConfigShortcutHelpersComponent>;
  let canReadSeeds: boolean;
  let canCreateEntity: boolean;
  let canCreateSeed: boolean;
  let canDeleteEntity: boolean;
  let canRunSeed: boolean;
  let canRunCrawlJob: boolean;

  beforeEach(async () => {
    canReadSeeds = true;
    canCreateEntity = false;
    canCreateSeed = false;
    canDeleteEntity = false;
    canRunSeed = false;
    canRunCrawlJob = false;

    await TestBed.configureTestingModule({
      imports: [ConfigShortcutHelpersComponent],
      providers: [
        provideMaterialAnimationsDisabled(),
        provideRouter([]),
        {
          provide: AbilityServiceSignal,
          useValue: {
            can: (action: string, subject: string) =>
              (action === 'read' && subject === Kind[Kind.SEED] && canReadSeeds)
              || (action === 'read' && [
                Kind[Kind.COLLECTION],
                Kind[Kind.BROWSERCONFIG],
                Kind[Kind.POLITENESSCONFIG],
              ].includes(subject))
              || (action === 'create' && subject === Kind[Kind.CRAWLENTITY] && canCreateEntity)
              || (action === 'create' && subject === Kind[Kind.SEED] && canCreateSeed)
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
    const links = Array.from<HTMLAnchorElement>(fixture.nativeElement.querySelectorAll('a'));
    const showSeedsLink = links.find(link => link.textContent.includes('Show seeds'));

    expect(showSeedsLink).toBeDefined();
    expect(showSeedsLink.getAttribute('href')).toContain('seed?entity_id=entity-1');
    expect(showSeedsLink.classList.contains('mat-mdc-chip')).toBe(true);
    expect(showSeedsLink.closest('mat-chip-set').getAttribute('aria-label'))
      .toBe('Configuration filters');
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

  it('puts Clone and Delete in the top-right configuration actions menu', async () => {
    canCreateEntity = true;
    canCreateSeed = true;
    canDeleteEntity = true;
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    const entity = new ConfigObject({id: 'entity-1', kind: Kind.CRAWLENTITY});
    fixture.componentRef.setInput('configObject', entity);
    fixture.detectChanges();
    const cloned: ConfigObject[] = [];
    const deleted: ConfigObject[] = [];
    const createdFrom: ConfigObject[] = [];
    fixture.componentInstance.clone.subscribe(value => cloned.push(value));
    fixture.componentInstance.delete.subscribe(value => deleted.push(value));
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

  it('falls back to reference IDs when related option collections are unavailable', async () => {
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

    expect(fixture.nativeElement.textContent).toContain('collection-1');
    expect(fixture.nativeElement.textContent).toContain('browser-config-1');
    expect(fixture.nativeElement.textContent).toContain('politeness-1');
  });

  it('uses configuration type names for reference chips in related cards', async () => {
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
    fixture.componentRef.setInput('showReferenceKindLabels', true);
    fixture.detectChanges();
    await fixture.whenStable();

    const text = fixture.nativeElement.textContent;
    expect(text).toContain('Collection');
    expect(text).toContain('Browser config');
    expect(text).toContain('Politeness config');
    expect(text).not.toContain('collection-1');
    expect(text).not.toContain('browser-config-1');
    expect(text).not.toContain('politeness-1');
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
    expect(runMenuItem.textContent).toContain('Crawl seed');
    runMenuItem.click();

    expect(emitted).toEqual([seed]);
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
    expect(runMenuItem.textContent).toContain('Run crawljob');
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
    expect(runMenuItem.textContent).toContain('Crawl seed with this crawljob');
    runMenuItem.click();

    expect(emitted).toEqual([{seed, crawlJob}]);
  });

  it('does not authorize a related seed crawl with crawljob permission alone', () => {
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

    expect(fixture.nativeElement.querySelector('[data-testid="config-actions-menu"]')).toBeNull();
  });
});
