import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CollectionDialogComponent} from './collection-dialog.component';
import {UntypedFormBuilder} from '@angular/forms';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {MatIconModule} from '@angular/material/icon';
import {AnnotationComponent, FilesizeInputComponent, LabelComponent} from '../..';
import {LabelService} from '../../../services';
import {of} from 'rxjs';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {AuthService} from '../../../../../core';
import {CollectionMetaComponent} from '../../collection-meta/collection-meta.component';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('CollectionDialogComponent', () => {
  let component: CollectionDialogComponent;
  let fixture: ComponentFixture<CollectionDialogComponent>;

  const configobject = new ConfigObject({
    kind: Kind.COLLECTION
  });

  const MY_CONF: ConfigDialogData = {
    configObject: configobject,
    options: {}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [MatDialogModule, MatIconModule, NoopAnimationsModule],
      declarations: [CollectionDialogComponent, CollectionMetaComponent, FilesizeInputComponent, LabelComponent, AnnotationComponent],
      providers: [UntypedFormBuilder,
        ...provideCoreTesting,
        {
          provide: LabelService,
          useValue: {
            getLabelKeys: () => of([])
          }
        },
        {
          provide: AuthService, useValue: {
            canEdit: () => true,
            canUpdate: () => true
          }
        },
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CollectionDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
