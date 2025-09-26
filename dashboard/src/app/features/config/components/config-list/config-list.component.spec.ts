import {ComponentFixture, TestBed} from '@angular/core/testing';

import {ConfigListComponent} from './config-list.component';
import {KeyboardShortcutsModule} from 'ng-keyboard-shortcuts';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import { provideZonelessChangeDetection } from '@angular/core';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('ConfigListComponent', () => {
  let component: ConfigListComponent;
  let fixture: ComponentFixture<ConfigListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [ConfigListComponent],
      imports: [
        KeyboardShortcutsModule,
        NoopAnimationsModule
      ],
      providers: [
        ...provideCoreTesting,
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
