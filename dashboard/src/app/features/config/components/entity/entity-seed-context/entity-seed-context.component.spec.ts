import {ComponentFixture, TestBed} from '@angular/core/testing';
import {By} from '@angular/platform-browser';
import {MatChipOptionHarness} from '@angular/material/chips/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {provideRouter} from '@angular/router';
import {EMPTY, of} from 'rxjs';

import {AuthService} from '../../../../../core';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {ConfigQuery} from '../../../../../shared/func';
import {ConfigObject, ConfigRef, Kind, Label, ListDataSource, Meta, Seed} from '../../../../../shared/models';
import {ConfigListComponent} from '../../config-list/config-list.component';
import {EntitySeedContextComponent} from './entity-seed-context.component';

describe('EntitySeedContextComponent', () => {
  let fixture: ComponentFixture<EntitySeedContextComponent>;
  let component: EntitySeedContextComponent;
  const entity = new ConfigObject({id: 'entity-1', kind: Kind.CRAWLENTITY});

  function dataSource(rows: ConfigObject[] = []): ListDataSource<ConfigObject, ConfigQuery> {
    return {
      rows$: of(rows),
      reset$: EMPTY,
      completed$: EMPTY,
      appendLoading$: of(false),
      appendFailed$: of(false),
      snapshot: rows,
      retry: vi.fn(),
      loadMore: vi.fn(),
    } as unknown as ListDataSource<ConfigObject, ConfigQuery>;
  }

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [EntitySeedContextComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {
          provide: AuthService,
          useValue: {
            canCreate: () => true,
            canUpdate: () => true,
            canDelete: () => true,
            canRead: () => true,
            canRunCrawl: () => true,
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(EntitySeedContextComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('entity', entity);
    fixture.componentRef.setInput('seedDataSource', dataSource());
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('shows seeds with the shared configuration list and the full seed menu', async () => {
    const seeds = [
      new ConfigObject({
        id: 'seed-one',
        kind: Kind.SEED,
        meta: new Meta({
          name: 'https://first.example/',
          description: 'Hidden seed description',
          labelList: [new Label({key: 'hidden', value: 'label'})],
        }),
        seed: new Seed({
          entityRef: new ConfigRef({kind: Kind.CRAWLENTITY, id: entity.id}),
          jobRefList: [new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-1'})],
        }),
      }),
      new ConfigObject({
        id: 'seed-two',
        kind: Kind.SEED,
        meta: new Meta({name: 'https://second.example/'}),
      }),
    ];
    fixture.componentRef.setInput('seedDataSource', dataSource(seeds));
    fixture.detectChanges();
    await fixture.whenStable();

    const rows = [...fixture.nativeElement.querySelectorAll('mat-list-item.item-row')] as HTMLElement[];
    const links = [...fixture.nativeElement.querySelectorAll('a.primary-detail-link')] as HTMLAnchorElement[];
    expect(links.map(link => link.textContent.trim())).toEqual([
      'https://first.example/',
      'https://second.example/',
    ]);
    expect(rows[0].querySelector('[matListItemLine]')).toBeNull();
    expect(rows[0].querySelector('.label-region')).toBeNull();
    expect(rows[0].querySelector('.config-kind-icon')).toBeNull();
    expect(rows[0].textContent).not.toContain('Hidden seed description');
    expect(rows[0].textContent).not.toContain('hidden');
    expect(getComputedStyle(fixture.nativeElement.querySelector('.seed-list mat-list')).paddingLeft).toBe('0px');
    expect(links.map(link => link.getAttribute('href'))).toEqual([
      'https://first.example/',
      'https://second.example/',
    ]);
    expect(links.map(link => link.getAttribute('target'))).toEqual(['_blank', '_blank']);
    expect(links.map(link => link.getAttribute('rel'))).toEqual(['noopener', 'noopener']);

    const opened: ConfigObject[] = [];
    component.openSeed.subscribe(seed => opened.push(seed));
    const list = fixture.debugElement.query(By.directive(ConfigListComponent)).componentInstance as ConfigListComponent;
    const linkClick = new MouseEvent('click', {cancelable: true});
    list.onPrimaryLink(seeds[0], linkClick);
    expect(linkClick.defaultPrevented).toBe(false);
    expect(opened).toEqual([]);

    rows[0].click();
    expect(opened).toEqual([seeds[0]]);

    const moreActions = rows[0].querySelector('button[aria-label="More actions"]') as HTMLButtonElement;
    moreActions.click();
    fixture.detectChanges();
    await fixture.whenStable();
    const menuText = (document.querySelector('.mat-mdc-menu-panel') as HTMLElement).textContent;
    for (const action of [
      'Show crawl executions',
      'Show seeds with the same crawl job',
      'Show seeds with the same entity',
      'Run crawl',
      'Edit',
      'Clone',
      'Delete',
    ]) {
      expect(menuText).toContain(action);
    }
    const navigationIcons = [...document.querySelectorAll<HTMLElement>('.mat-mdc-menu-panel a mat-icon')]
      .map(icon => icon.textContent.trim());
    expect(navigationIcons).toEqual(['hdr_weak', 'link', 'link']);
  });

  it('emits seed state changes', async () => {
    const changes: (boolean | null)[] = [];
    component.seedStatusChange.subscribe(value => changes.push(value));
    const stateChips = await TestbedHarnessEnvironment.loader(fixture).getAllHarnesses(
      MatChipOptionHarness.with({ancestor: '.seed-list-heading'}),
    );

    await stateChips[1].select();

    expect(changes).toEqual([true]);
  });

  it('puts the create seed action before the title and emits the entity', () => {
    const created: ConfigObject[] = [];
    component.createSeed.subscribe(value => created.push(value));
    const titleActions = fixture.nativeElement.querySelector('.seed-title-actions') as HTMLElement;
    const headingRow = fixture.nativeElement.querySelector('.seed-list-heading') as HTMLElement;
    const button = titleActions.querySelector('button.create-seed-fab') as HTMLButtonElement;
    const title = titleActions.querySelector('h2') as HTMLElement;
    const stateFilters = headingRow.querySelector('mat-chip-listbox') as HTMLElement;

    expect(button.compareDocumentPosition(title) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(getComputedStyle(button).boxShadow).toBe('none');
    expect(title.textContent.trim()).toBe('Seeds');
    expect(getComputedStyle(titleActions).alignItems).toBe('center');
    expect(getComputedStyle(headingRow).alignItems).toBe('center');
    expect(stateFilters).not.toBeNull();
    button.click();
    expect(created).toEqual([entity]);
  });
});
