import {SimpleChange} from '@angular/core';
import {TestBed} from '@angular/core/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {MatDialog} from '@angular/material/dialog';
import {MatCheckboxHarness} from '@angular/material/checkbox/testing';
import {of} from 'rxjs';

import {AuthService} from '../../../../../core';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {ConfigObject, ConfigRef, Kind, Meta, Seed} from '../../../../../shared/models';
import {MoveSeedDialogComponent} from '../move-seed-dialog/move-seed-dialog.component';
import {SeedDetailsComponent} from './seed-details.component';

describe('SeedDetailsComponent move flow', () => {
  const destination = new ConfigRef({kind: Kind.CRAWLENTITY, id: 'destination'});
  const dialogOpen = vi.fn(() => ({afterClosed: () => of(destination)}));

  beforeEach(async () => {
    dialogOpen.mockClear();
    await TestBed.configureTestingModule({
      imports: [SeedDetailsComponent],
      providers: [
        {provide: AuthService, useValue: {canUpdate: () => true, canDelete: () => true}},
        {provide: MatDialog, useValue: {open: dialogOpen}},
      ],
    })
      .overrideComponent(SeedDetailsComponent, {set: {template: ''}})
      .compileComponents();
  });

  it('opens the entity picker and emits an immediate move parcel', () => {
    const fixture = TestBed.createComponent(SeedDetailsComponent);
    const component = fixture.componentInstance;
    const seed = new ConfigObject({
      id: 'seed-1',
      kind: Kind.SEED,
      meta: new Meta({name: 'https://example.test'}),
      seed: new Seed({entityRef: new ConfigRef({kind: Kind.CRAWLENTITY, id: 'current'})}),
    });
    component.configObject = seed;
    component.crawlJobs = [];
    component.ngOnChanges({});
    const moved = vi.fn();
    component.move.subscribe(moved);

    component.onMoveSeed();

    expect(dialogOpen).toHaveBeenCalledWith(MoveSeedDialogComponent, expect.objectContaining({
      data: {seed},
    }));
    expect(moved).toHaveBeenCalledWith({seed, entityRef: destination});
  });

  it('does not open the picker while the seed form has unsaved changes', () => {
    const fixture = TestBed.createComponent(SeedDetailsComponent);
    const component = fixture.componentInstance;
    component.configObject = new ConfigObject({id: 'seed-1', kind: Kind.SEED});
    component.form.markAsDirty();

    component.onMoveSeed();

    expect(dialogOpen).not.toHaveBeenCalled();
    expect(component.moveSeedTooltip).toContain('Save or revert changes');
  });
});

describe('SeedDetailsComponent form', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SeedDetailsComponent],
      providers: [
        ...provideCoreTesting,
        {provide: AuthService, useValue: {canUpdate: () => true, canDelete: () => true}},
        {provide: MatDialog, useValue: {open: vi.fn()}},
      ],
    }).compileComponents();
  });

  it('renders Deactivated as a normal checkbox', async () => {
    const fixture = TestBed.createComponent(SeedDetailsComponent);
    const component = fixture.componentInstance;
    component.configObject = new ConfigObject({
      id: 'seed-1',
      kind: Kind.SEED,
      meta: new Meta({name: 'https://example.test'}),
      seed: new Seed({
        disabled: true,
        entityRef: new ConfigRef({kind: Kind.CRAWLENTITY, id: 'entity-1'}),
      }),
    });
    component.crawlJobs = [];
    component.ngOnChanges({
      configObject: new SimpleChange(null, component.configObject, true),
    });
    fixture.detectChanges();
    await fixture.whenStable();

    const checkbox = await TestbedHarnessEnvironment.loader(fixture).getHarness(MatCheckboxHarness);
    expect(await checkbox.getLabelText()).toBe('Deactivated');
    expect(await checkbox.isChecked()).toBe(true);
  });
});
