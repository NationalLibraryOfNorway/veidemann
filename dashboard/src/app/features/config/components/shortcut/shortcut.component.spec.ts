import {Clipboard} from '@angular/cdk/clipboard';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';

import {provideMaterialAnimationsDisabled} from '../../../../core/core.testing.module';
import {SnackBarService} from '../../../../core';
import {ConfigObject, ConfigRef, CrawlConfig, Kind} from '../../../../shared/models';
import {ConfigShortcutHelpersComponent} from './shortcut.component';

describe('ConfigShortcutHelpersComponent', () => {
  let fixture: ComponentFixture<ConfigShortcutHelpersComponent>;
  let canReadSeeds: boolean;
  let canCreateEntity: boolean;
  let canCreateSeed: boolean;
  let copy: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    canReadSeeds = true;
    canCreateEntity = false;
    canCreateSeed = false;
    copy = vi.fn(() => true);

    await TestBed.configureTestingModule({
      imports: [ConfigShortcutHelpersComponent],
      providers: [
        provideMaterialAnimationsDisabled(),
        provideRouter([]),
        {provide: Clipboard, useValue: {copy}},
        {provide: SnackBarService, useValue: {openSnackBar: vi.fn(), openError: vi.fn()}},
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
              || (action === 'create' && subject === Kind[Kind.SEED] && canCreateSeed),
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
    expect(fixture.nativeElement.querySelector('mat-chip-set').getAttribute('aria-label'))
      .toBe('Configuration shortcuts');
    expect(fixture.nativeElement.querySelector('.shortcut-group, .group-label')).toBeNull();
    expect(fixture.nativeElement.querySelector('.more-actions')).toBeNull();
  });

  it('shows the full main ID as a copyable chip and allows related cards to suppress it', () => {
    const idChip = fixture.nativeElement.querySelector('.main-id-chip') as HTMLElement;
    expect(idChip.textContent).toContain('entity-1');

    idChip.click();
    expect(copy).toHaveBeenCalledWith('entity-1');

    fixture.componentRef.setInput('showMainId', false);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.main-id-chip')).toBeNull();
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

  it('renders keyboard-operable action chips without native buttons', () => {
    canCreateEntity = true;
    canCreateSeed = true;
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    const entity = new ConfigObject({id: 'entity-1', kind: Kind.CRAWLENTITY});
    fixture.componentRef.setInput('configObject', entity);
    fixture.detectChanges();
    const cloned: ConfigObject[] = [];
    const createdFrom: ConfigObject[] = [];
    fixture.componentInstance.clone.subscribe(value => cloned.push(value));
    fixture.componentInstance.createSeed.subscribe(value => createdFrom.push(value));

    const actionChips = Array.from<HTMLElement>(
      fixture.nativeElement.querySelectorAll('mat-chip[role="button"]'),
    );
    const cloneChip = actionChips.find(chip => chip.textContent.includes('Clone'));
    const createSeedChip = actionChips.find(chip => chip.textContent.includes('Create Seed'));

    cloneChip.click();
    const enterEvent = new KeyboardEvent('keydown', {key: 'Enter', bubbles: true, cancelable: true});
    cloneChip.dispatchEvent(enterEvent);
    const spaceEvent = new KeyboardEvent('keydown', {key: ' ', bubbles: true, cancelable: true});
    cloneChip.dispatchEvent(spaceEvent);
    createSeedChip.click();

    expect(fixture.nativeElement.querySelector('button.mat-mdc-chip')).toBeNull();
    expect(fixture.nativeElement.querySelector('.shortcut-group, .group-label')).toBeNull();
    expect(actionChips.every(chip => chip.tabIndex === 0)).toBe(true);
    expect(enterEvent.defaultPrevented).toBe(true);
    expect(spaceEvent.defaultPrevented).toBe(true);
    expect(cloned).toEqual([entity, entity, entity]);
    expect(createdFrom).toEqual([entity]);
    expect(fixture.nativeElement.querySelector('[mat-menu-item]')).toBeNull();
  });

  it('renders the opt-in script editor action in the shortcut chip set', () => {
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'script-1',
      kind: Kind.BROWSERSCRIPT,
    }));
    fixture.componentRef.setInput('showScriptEditorAction', true);
    fixture.detectChanges();
    let opened = 0;
    fixture.componentInstance.openScriptEditor.subscribe(() => opened++);

    const chipSet = fixture.nativeElement.querySelector('mat-chip-set') as HTMLElement;
    const editChip = Array.from<HTMLElement>(chipSet.querySelectorAll('mat-chip[role="button"]'))
      .find(chip => chip.textContent.includes('Edit script'));

    expect(editChip).toBeDefined();
    editChip.click();
    const enterEvent = new KeyboardEvent('keydown', {key: 'Enter', bubbles: true, cancelable: true});
    editChip.dispatchEvent(enterEvent);

    expect(opened).toBe(2);
    expect(enterEvent.defaultPrevented).toBe(true);
  });

  it('does not render the script editor action unless enabled', () => {
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigShortcutHelpersComponent);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'script-1',
      kind: Kind.BROWSERSCRIPT,
    }));
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).not.toContain('Edit script');
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
    fixture.componentRef.setInput('showMainId', false);
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
});
