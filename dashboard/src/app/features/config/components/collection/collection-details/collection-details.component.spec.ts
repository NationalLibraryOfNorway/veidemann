import {SimpleChange} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {of} from 'rxjs';

import {AuthService} from '../../../../../core';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {Collection, ConfigObject, Kind, Meta, rotationPolicies, RotationPolicy, subCollectionTypes} from '../../../../../shared/models';
import {LabelService} from '../../../services';
import {CollectionDetailsComponent} from './collection-details.component';
import {MatCheckboxHarness} from '@angular/material/checkbox/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';

describe('CollectionDetailsComponent', () => {
  let fixture: ComponentFixture<CollectionDetailsComponent>;
  let component: CollectionDetailsComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CollectionDetailsComponent],
      providers: [
        ...provideCoreTesting,
        {provide: AuthService, useValue: {canUpdate: () => true, canDelete: () => true}},
        {provide: LabelService, useValue: {getLabelKeys: () => of([])}},
      ],
    }).compileComponents();
    fixture = TestBed.createComponent(CollectionDetailsComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject({
      id: 'collection-1', kind: Kind.COLLECTION, meta: new Meta({name: 'Collection'}),
      collection: new Collection({compress: true, fileSize: 1024}),
    });
    component.rotationPolicies = rotationPolicies;
    component.subCollectionTypes = subCollectionTypes;
    component.ngOnChanges({configObject: new SimpleChange(null, component.configObject, true)});
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('renders saved details with copy, rotation selects, compression checkbox, and subcollection add chips', async () => {
    expect(component).toBeTruthy();
    expect(fixture.nativeElement.querySelector('button[aria-label="Copy ID"]')).not.toBeNull();
    expect(fixture.nativeElement.querySelectorAll('mat-select').length).toBe(2);
    const checkbox = await TestbedHarnessEnvironment.loader(fixture).getHarness(MatCheckboxHarness);
    expect(await checkbox.getLabelText()).toBe('Compressed');
    expect(await checkbox.isChecked()).toBe(true);
    expect(fixture.nativeElement.textContent).toContain('Add screenshot collection');
    expect(fixture.nativeElement.textContent).toContain('Add DNS collection');
    expect(fixture.nativeElement.querySelector('fieldset.mat-elevation-z1')).toBeNull();
  });

  it('marks the form dirty when the compression checkbox changes and saves the value', async () => {
    const checkbox = await TestbedHarnessEnvironment.loader(fixture).getHarness(MatCheckboxHarness);
    await checkbox.toggle();
    expect(await checkbox.isChecked()).toBe(false);
    expect(component.canUpdate).toBe(true);
    let update: ConfigObject | undefined;
    component.update.subscribe(value => update = value);
    component.onUpdate();
    expect(update?.collection.compress).toBe(false);
  });

  it('reverts confirmed changes', () => {
    component.form.get('collectionDedupPolicy').setValue(RotationPolicy.YEARLY);
    component.form.get('collectionDedupPolicy').markAsDirty();
    component.onRevert();
    expect(component.form.get('collectionDedupPolicy').value).toBe(RotationPolicy.NONE);
    expect(component.form.pristine).toBe(true);
  });

  it('adds an inline subcollection and invalidates the form until it has a valid name', () => {
    const addScreenshot = fixture.nativeElement.querySelector('[data-testid="addScreenshotSubcollectionButton"]');
    addScreenshot.click();
    fixture.detectChanges();

    expect(component.form.dirty).toBe(true);
    expect(component.form.invalid).toBe(true);
    expect(fixture.nativeElement.querySelector('[data-testid="addScreenshotSubcollectionButton"]')).toBeNull();

    const name = fixture.nativeElement.querySelector('[data-testid="subcollectionName"]');
    name.value = 'screenshots';
    name.dispatchEvent(new Event('input', {bubbles: true}));
    fixture.detectChanges();
    expect(component.form.valid).toBe(true);
  });

});
