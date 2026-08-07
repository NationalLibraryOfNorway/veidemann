import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {of} from 'rxjs';

import {provideMaterialAnimationsDisabled} from '../../../../../core/core.testing.module';
import {ConfigObject, ConfigRef, Kind, Meta, Seed} from '../../../../../shared/models';
import {ConfigService} from '../../../../../shared/services';
import {
  MoveSeedDialogComponent,
  MoveSeedDialogData,
} from './move-seed-dialog.component';

describe('MoveSeedDialogComponent', () => {
  let fixture: ComponentFixture<MoveSeedDialogComponent>;
  const currentEntity = new ConfigRef({kind: Kind.CRAWLENTITY, id: 'current-entity'});
  const seed = new ConfigObject({
    id: 'seed-1',
    kind: Kind.SEED,
    meta: new Meta({name: 'https://example.test'}),
    seed: new Seed({entityRef: currentEntity}),
  });
  const entities = [
    new ConfigObject({id: currentEntity.id, kind: Kind.CRAWLENTITY, meta: new Meta({name: 'Current'})}),
    new ConfigObject({
      id: 'destination-entity',
      kind: Kind.CRAWLENTITY,
      meta: new Meta({name: 'Destination', description: 'The destination entity'}),
    }),
  ];
  const search = vi.fn(() => of(...entities));

  beforeEach(async () => {
    search.mockClear();
    vi.useFakeTimers();
    await TestBed.configureTestingModule({
      imports: [MoveSeedDialogComponent],
      providers: [
        provideMaterialAnimationsDisabled(),
        {provide: MAT_DIALOG_DATA, useValue: {seed} satisfies MoveSeedDialogData},
        {provide: MatDialogRef, useValue: {}},
        {provide: ConfigService, useValue: {search}},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(MoveSeedDialogComponent);
    fixture.detectChanges();
  });

  afterEach(() => vi.useRealTimers());

  it('loads alphabetized entities initially and excludes the current entity', async () => {
    await vi.advanceTimersByTimeAsync(250);
    fixture.detectChanges();

    expect(search).toHaveBeenCalledWith(expect.objectContaining({
      kind: Kind.CRAWLENTITY,
      term: null,
      active: 'name',
      direction: 'asc',
    }), {offset: 0, pageSize: 50});
    expect(fixture.nativeElement.textContent).toContain('Destination');
    expect(fixture.nativeElement.textContent).toContain('The destination entity');
    expect(fixture.nativeElement.textContent).not.toContain('Current');
  });

  it('debounces entity-name searches and returns the selected entity reference', async () => {
    await vi.advanceTimersByTimeAsync(250);
    fixture.componentInstance.search.setValue('  dest  ');
    await vi.advanceTimersByTimeAsync(249);
    expect(search).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(1);
    expect(search).toHaveBeenLastCalledWith(expect.objectContaining({term: 'dest'}), {
      offset: 0,
      pageSize: 50,
    });

    fixture.componentInstance.selectedEntity = entities[1];
    expect(fixture.componentInstance.result()).toEqual(expect.objectContaining({
      kind: Kind.CRAWLENTITY,
      id: 'destination-entity',
    }));
  });

  it('shows an empty state when no destination entities match', async () => {
    search.mockReturnValueOnce(of());
    await vi.advanceTimersByTimeAsync(250);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('[role="status"]').textContent)
      .toContain('No matching entities found.');
  });
});
