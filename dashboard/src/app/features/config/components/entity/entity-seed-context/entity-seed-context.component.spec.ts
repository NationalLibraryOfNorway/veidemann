import {ComponentFixture, TestBed} from '@angular/core/testing';
import {By} from '@angular/platform-browser';
import {MatButtonHarness} from '@angular/material/button/testing';
import {MatMenuHarness} from '@angular/material/menu/testing';
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

  it('uses the shared order and state controls', async () => {
    const changes: (boolean | null)[] = [];
    const sorts: {active: string; direction: string}[] = [];
    component.seedStatusChange.subscribe(value => changes.push(value));
    component.seedSortChange.subscribe(value => sorts.push(value));
    const loader = TestbedHarnessEnvironment.loader(fixture);
    const buttons = await loader.getAllHarnesses(MatButtonHarness.with({ancestor: '.list-header-controls'}));

    expect([...fixture.nativeElement.querySelectorAll('.list-header-controls button mat-icon')]
      .map((icon: HTMLElement) => icon.textContent.trim())).toEqual([
      'sort',
      'toggle_off',
      'toggle_on',
    ]);

    await buttons[1].click();
    await buttons[2].click();
    expect(changes).toEqual([true, false]);

    const menu = await loader.getHarness(MatMenuHarness.with({triggerIconName: 'sort'}));
    await menu.open();
    await menu.clickItem({text: 'Name: A–Z'});

    expect(sorts).toEqual([{active: 'name', direction: 'asc'}]);
  });

  it('renders a related-context heading and right-aligns the create action in the controls row', () => {
    const created: ConfigObject[] = [];
    component.createSeed.subscribe(value => created.push(value));
    const title = fixture.nativeElement.querySelector('h3#entity-seeds-title') as HTMLElement;
    const seedList = fixture.nativeElement.querySelector('.seed-list') as HTMLElement;
    const headingRow = fixture.nativeElement.querySelector('.list-header-row') as HTMLElement;
    const controls = headingRow.querySelector('.list-header-controls') as HTMLElement;
    const trailing = headingRow.querySelector('.header-trailing') as HTMLElement;
    const button = trailing.querySelector('button.create-seed-fab') as HTMLButtonElement;

    expect(fixture.nativeElement.querySelector('h2#entity-seeds-title')).toBeNull();
    expect(fixture.nativeElement.querySelector('mat-chip-listbox')).toBeNull();
    expect(getComputedStyle(button).boxShadow).toBe('none');
    expect(title.textContent.trim()).toBe('Seeds');
    expect(getComputedStyle(seedList).borderTopWidth).toBe('1px');
    expect(getComputedStyle(seedList).borderTopStyle).toBe('solid');
    expect(controls.compareDocumentPosition(trailing) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(getComputedStyle(trailing).justifyContent).toBe('flex-end');
    expect(Math.abs(controls.getBoundingClientRect().top - button.getBoundingClientRect().top))
      .toBeLessThanOrEqual(4);
    button.click();
    expect(created).toEqual([entity]);
  });
});
