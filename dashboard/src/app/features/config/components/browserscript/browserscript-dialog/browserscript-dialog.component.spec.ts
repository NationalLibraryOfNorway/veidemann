import {ComponentFixture, TestBed} from '@angular/core/testing';

import {BrowserScriptDialogComponent} from './browserscript-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {CommonsModule} from '../../../../commons';
import {MetaComponent} from '../../meta/meta.component';
import {LabelComponent} from '../../label/label.component';
import {LabelService} from '../../../services';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {of} from 'rxjs';
import {AnnotationComponent} from '../../annotation/annotation.component';
import {CoreTestingModule} from '../../../../core/core.testing.module';
import {AuthService} from '../../../../core';
import {MonacoEditorModule} from 'ngx-monaco-editor-v2';

describe('BrowserScriptDialogComponent', () => {
  let component: BrowserScriptDialogComponent;
  let fixture: ComponentFixture<BrowserScriptDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({kind: Kind.BROWSERSCRIPT}),
    options: {}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        CoreTestingModule.forRoot(),
        CommonsModule,
        NoopAnimationsModule,
        MonacoEditorModule.forRoot()
      ],
      providers: [
        {
          provide: LabelService,
          useValue: {
            getLabelKeys: () => of([])
          }
        },
        {
          provide: AuthService,
          useValue: {
            canUpdate: () => true
          }
        },
        {
          provide: MAT_DIALOG_DATA,
          useValue: MY_CONF
        },
        {
          provide: MatDialogRef,
          useValue: {}
        }
      ],
      declarations: [BrowserScriptDialogComponent, MetaComponent, LabelComponent, AnnotationComponent]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(BrowserScriptDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
