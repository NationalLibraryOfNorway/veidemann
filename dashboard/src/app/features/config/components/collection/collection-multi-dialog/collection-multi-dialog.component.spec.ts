import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {Collection, ConfigObject, Kind, Label, Meta, RotationPolicy} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {CollectionMultiDialogComponent} from './collection-multi-dialog.component';

describe('CollectionMultiDialogComponent', () => {
  let fixture: ComponentFixture<CollectionMultiDialogComponent>;
  let component: CollectionMultiDialogComponent;

  const data: ConfigDialogData = {
    configObject: new ConfigObject({
      kind: Kind.COLLECTION,
      collection: new Collection({
        collectionDedupPolicy: RotationPolicy.DAILY,
        fileRotationPolicy: RotationPolicy.MONTHLY,
        compress: true,
        fileSize: 1024,
      }),
    }),
    options: {rotationPolicies: [RotationPolicy.DAILY, RotationPolicy.MONTHLY]},
    allSelected: false,
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CollectionMultiDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: data},
        {provide: MatDialogRef, useValue: {}},
      ],
    }).compileComponents();
    fixture = TestBed.createComponent(CollectionMultiDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('updates only explicitly selected scalar fields', () => {
    component.collectionDedupPolicy.setValue(RotationPolicy.MONTHLY);
    component.collectionDedupPolicy.markAsDirty();
    const compress = fixture.nativeElement.querySelector(
      'app-boolean-override button'
    ) as HTMLButtonElement;
    compress.click();

    const result = component.onDialogClose();
    expect(result.pathList).toEqual([
      'collection.collectionDedupPolicy',
      'collection.compress',
    ]);
    expect(result.updateTemplate.collection.collectionDedupPolicy).toBe(RotationPolicy.MONTHLY);
    expect(result.updateTemplate.collection.compress).toBe(true);
  });

  it('clears the compression override when the selected button is clicked again', () => {
    const compress = fixture.nativeElement.querySelector(
      'app-boolean-override button'
    ) as HTMLButtonElement;
    compress.click();
    compress.click();

    expect(component.canUpdate).toBe(false);
    expect(component.onDialogClose().pathList).not.toContain('collection.compress');
  });

  it('does not expose subcollections', () => {
    expect(fixture.nativeElement.querySelector('app-subcollection-chips')).toBeNull();
  });

  it('merges common collection labels and scalar values', () => {
    const first = new ConfigObject({
      kind: Kind.COLLECTION,
      meta: new Meta({labelList: [new Label({key: 'owner', value: 'news'})]}),
      collection: new Collection({compress: true, fileSize: 1024}),
    });
    const second = new ConfigObject({
      kind: Kind.COLLECTION,
      meta: new Meta({labelList: [new Label({key: 'owner', value: 'news'})]}),
      collection: new Collection({compress: false, fileSize: 1024}),
    });

    const merged = ConfigObject.mergeConfigs([first, second]);
    expect(merged.meta.labelList).toEqual([{key: 'owner', value: 'news'}]);
    expect(merged.collection.fileSize).toBe(1024);
    expect(merged.collection.compress).toBeNull();
  });
});
