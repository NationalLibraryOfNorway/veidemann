import {ComponentFixture, TestBed} from '@angular/core/testing';

import {ConfigListComponent} from './config-list.component';
import {KeyboardShortcutsModule} from 'ng-keyboard-shortcuts';
import {MaterialModule} from '../../../commons/material.module';
import {CommonsModule} from '../../../commons';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import { provideZonelessChangeDetection } from '@angular/core';

describe('ConfigListComponent', () => {
  let component: ConfigListComponent;
  let fixture: ComponentFixture<ConfigListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [ConfigListComponent],
      imports: [
        KeyboardShortcutsModule,
        MaterialModule,
        CommonsModule,
        NoopAnimationsModule
      ],
      providers: [
        provideZonelessChangeDetection(),
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(ConfigListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
